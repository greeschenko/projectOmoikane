package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCloudEventRoundTrip verifies the CloudEvents envelope marshals and
// unmarshals with the exact spec field names a Kafka consumer will see.
func TestCloudEventRoundTrip(t *testing.T) {
	ev := CloudEvent{
		SpecVersion: "1.0",
		ID:          "evt_123",
		Source:      SourceContent,
		Type:        TypePostPublished,
		Subject:     "post/42",
		Time:        time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC),
		Data:        json.RawMessage(`{"id":42,"status":"published"}`),
	}

	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify the exact CloudEvents 1.0 wire names are used.
	for _, key := range []string{"specversion", "id", "source", "type", "subject", "time", "data"} {
		if !json.Valid(b) || !strings.Contains(string(b), `"`+key+`"`) {
			t.Fatalf("envelope missing key %q: %s", key, b)
		}
	}

	var got CloudEvent
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != ev.ID || got.Type != ev.Type || got.Subject != ev.Subject || !got.Time.Equal(ev.Time) {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

// TestSchemaCatalogParses ensures every JSON Schema in schemas/ is valid JSON.
// The catalog is the payload contract for each event type.
func TestSchemaCatalogParses(t *testing.T) {
	dir := filepath.Join("schemas")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read schemas dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("schema catalog is empty")
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: read: %v", path, err)
		}
		if !json.Valid(raw) {
			t.Fatalf("%s: invalid JSON", path)
		}
		var v map[string]any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("%s: unmarshal: %v", path, err)
		}
		if v["$id"] == "" || v["type"] == nil || v["properties"] == nil {
			t.Fatalf("%s: missing required schema keys ($id/type/properties)", path)
		}
	}
}

// TestSchemaCatalogIsComplete ensures every platform event Type constant has a
// JSON Schema file in the catalog (single source of truth for the contract).
func TestSchemaCatalogIsComplete(t *testing.T) {
	dir := filepath.Join("schemas")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read schemas dir: %v", err)
	}

	// Map schema file name (sans .json) to its declared $id.
	ids := map[string]bool{}
	byFile := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: read: %v", path, err)
		}
		var v struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("%s: unmarshal: %v", path, err)
		}
		file := strings.TrimSuffix(e.Name(), ".json")
		byFile[file] = v.ID
		ids[v.ID] = true
	}

	// Every type constant must have a catalog entry whose $id matches Type.
	types := map[string]string{
		TypeUserRegistered:  "user.registered.json",
		TypeAuthLogin:       "auth.login.json",
		TypePagePublished:   "page.published.json",
		TypePageUpdated:     "page.updated.json",
		TypePostPublished:   "post.published.json",
		TypePostUpdated:     "post.updated.json",
		TypeMediaUploaded:   "media.uploaded.json",
		TypeMessageCreated:  "message.created.json",
		TypeContactReceived: "contact.received.json",
		TypeSettingsUpdated: "settings.updated.json",
	}
	for typ, file := range types {
		id, ok := byFile[strings.TrimSuffix(file, ".json")]
		if !ok {
			t.Errorf("no schema file for type %q (wanted %s)", typ, file)
			continue
		}
		if id != typ {
			t.Errorf("schema %s declares $id %q, expected %q (type constant)", file, id, typ)
		}
	}
}

// TestNewEventID ensures event ids are unique, hex-encoded and 32 chars long
// (16 random bytes) — the CloudEvents "id" must be unique per source.
func TestNewEventID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewEventID()
		if len(id) != 32 {
			t.Fatalf("NewEventID length = %d, want 32 (16 bytes hex)", len(id))
		}
		for _, c := range id {
			if !strings.ContainsRune("0123456789abcdef", c) {
				t.Fatalf("NewEventID %q contains non-hex char %q", id, c)
			}
		}
		if seen[id] {
			t.Fatalf("NewEventID returned duplicate %q", id)
		}
		seen[id] = true
	}
}
