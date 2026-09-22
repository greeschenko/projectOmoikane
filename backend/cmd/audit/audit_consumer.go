package main

import (
	"context"
	"encoding/json"
	"fmt"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// auditEventHandler is the audit service's Kafka consumer handler (Phase 32).
// It maps CloudEvents from the omoikane.events backbone to AuditLog rows.
//
// Delivery is at-least-once end to end, so Handle must be idempotent:
// AuditLog.EventID (the CloudEvent id) is unique-indexed and a redelivered
// event is skipped — the duplicate is treated as success so the consumer
// commits the offset instead of DLQ-ing it.
type auditEventHandler struct {
	db *gorm.DB
}

// Handle stores one CloudEvent as an audit log row. Events with no audit
// mapping (unknown or out-of-scope types) are ignored with nil error so the
// backbone keeps flowing; a mapped type with an unparseable payload is a real
// error (the consumer retries then dead-letters the poison message).
func (h *auditEventHandler) Handle(ctx context.Context, ev events.CloudEvent) error {
	entry, mapped, err := mapEventToLog(ev)
	if err != nil {
		return fmt.Errorf("audit: map %s event %s: %w", ev.Type, ev.ID, err)
	}
	if !mapped {
		return nil
	}
	if err := h.db.Create(&entry).Error; err != nil {
		if h.alreadyStored(entry.EventID) {
			return nil
		}
		return fmt.Errorf("audit: store %s event %s: %w", ev.Type, ev.ID, err)
	}
	return nil
}

// alreadyStored reports whether an event id already produced a row (at-least-
// once redelivery guard behind the unique index).
func (h *auditEventHandler) alreadyStored(eventID string) bool {
	var n int64
	h.db.Model(&models.AuditLog{}).Where("event_id = ?", eventID).Count(&n)
	return n > 0
}

// Event payload contracts. These mirror internal/events/schemas/*.json; they
// are the subset the audit service cares about. Unknown fields are ignored by
// encoding/json, so producers may extend payloads without breaking the audit
// mapping (forward-compatible).
type userRegisteredPayload struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type authLoginPayload struct {
	ID     uint   `json:"id"`
	Email  string `json:"email"`
	Method string `json:"method"`
}

type pagePublishedPayload struct {
	ID    uint   `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type postPublishedPayload struct {
	ID    uint   `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type mediaUploadedPayload struct {
	ID       uint   `json:"id"`
	Filename string `json:"filename"`
	Alt      string `json:"alt"`
	Size     int64  `json:"size"`
}

type contactReceivedPayload struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
}

type settingsUpdatedPayload struct {
	SiteName    string `json:"siteName"`
	BlogEnabled bool   `json:"blogEnabled"`
}

// mapEventToLog converts a CloudEvent into an AuditLog row. Returns:
//
//   - (entry, true, nil) for a mapped, well-formed event;
//   - (zero, false, nil) for a type with no audit mapping (ignored, not
//     retried);
//   - (zero, false, err) for a MAPPED type whose payload is malformed — a
//     poison message the consumer retries then dead-letters.
//
// The action vocabulary matches the admin UI chip colors in
// frontend/app/admin/audit-logs/page.tsx.
func mapEventToLog(ev events.CloudEvent) (models.AuditLog, bool, error) {
	entry := models.AuditLog{
		EventID: ev.ID,
		// Subject conveys no actor — Phase 32 events carry entity identity,
		// not the acting user. Row user fields are filled where the payload
		// carries an identity (register/login/contact) and default to "system"
		// otherwise (documented follow-up: actor meta in event payloads).
		UserName: "system",
	}

	switch ev.Type {
	case events.TypeUserRegistered:
		var p userRegisteredPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.UserID = p.ID
		entry.UserName = p.Email
		entry.Action = "register"
		entry.EntityType = "user"
		entry.EntityID = p.ID
		entry.Detail = fmt.Sprintf("Registered user %s (role %s)", p.Email, p.Role)

	case events.TypeAuthLogin:
		var p authLoginPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.UserID = p.ID
		entry.UserName = p.Email
		entry.Action = "login"
		entry.EntityType = "user"
		entry.EntityID = p.ID
		entry.Detail = fmt.Sprintf("Logged in via %s", p.Method)

	case events.TypePagePublished:
		var p pagePublishedPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.Action = "publish"
		entry.EntityType = "page"
		entry.EntityID = p.ID
		entry.Detail = fmt.Sprintf("Published page %q (/%s)", p.Title, p.Slug)

	case events.TypePostPublished:
		var p postPublishedPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.Action = "publish"
		entry.EntityType = "post"
		entry.EntityID = p.ID
		entry.Detail = fmt.Sprintf("Published post %q (/%s)", p.Title, p.Slug)

	case events.TypeMediaUploaded:
		var p mediaUploadedPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.Action = "upload"
		entry.EntityType = "media"
		entry.EntityID = p.ID
		entry.Detail = fmt.Sprintf("Uploaded media %q (%d bytes)", p.Filename, p.Size)

	case events.TypeContactReceived:
		var p contactReceivedPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.UserID = 0 // public submission — anonymous submitter
		entry.UserName = p.Name
		entry.Action = "contact"
		entry.EntityType = "contact"
		entry.EntityID = p.ID
		entry.Detail = fmt.Sprintf("Contact message from %s <%s>: %s", p.Name, p.Email, p.Subject)

	case events.TypeSettingsUpdated:
		var p settingsUpdatedPayload
		if err := decodePayload(ev, &p); err != nil {
			return entry, false, err
		}
		entry.Action = "update"
		entry.EntityType = "settings"
		entry.EntityID = 1
		entry.Detail = fmt.Sprintf("Updated site settings (siteName=%s, blogEnabled=%t)", p.SiteName, p.BlogEnabled)

	default:
		// Unknown/out-of-scope event type — no audit row, not an error.
		return entry, false, nil
	}

	return entry, true, nil
}

// decodePayload unmarshals the CloudEvent data bytes into a payload struct.
// Unknown fields are ignored (producers stay forward-compatible); only
// syntactically invalid JSON is an error.
func decodePayload(ev events.CloudEvent, out any) error {
	if len(ev.Data) == 0 {
		return fmt.Errorf("empty payload")
	}
	return json.Unmarshal(ev.Data, out)
}
