package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// InternalStatsUsers returns the auth-owned slice of the dashboard stats:
// user count plus the 7-day zero-filled registrations chart (Phase 26 shape).
// Called by the dashboard facade.
// @Summary Auth dashboard stats (internal)
// @Description Returns the user count and last-7-days registrations used by the dashboard facade.
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /internal/stats [get]
func (h *Handler) InternalStatsUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var userCount int64
	h.DB.Model(&models.User{}).Count(&userCount)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"users":               userCount,
		"recentRegistrations": last7DaysRegistrations(h.DB),
	})
}

// InternalStatsContent returns the content-owned slice of the dashboard stats:
// page and post (blog) counts. Called by the dashboard facade.
// @Summary Content dashboard stats (internal)
// @Description Returns the page and blog post counts used by the dashboard facade.
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /internal/stats [get]
func (h *Handler) InternalStatsContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var pageCount int64
	var postCount int64
	h.DB.Model(&models.Page{}).Count(&pageCount)
	h.DB.Model(&models.BlogPost{}).Count(&postCount)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": pageCount,
		"posts": postCount,
	})
}

// InternalStatsMedia returns the media-owned slice of the dashboard stats.
// Called by the dashboard facade.
// @Summary Media dashboard stats (internal)
// @Description Returns the media item count used by the dashboard facade.
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /internal/stats [get]
func (h *Handler) InternalStatsMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var mediaCount int64
	h.DB.Model(&models.MediaItem{}).Count(&mediaCount)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"media": mediaCount,
	})
}

// InternalStatsMessages returns the messages-owned slice of the dashboard
// stats: message count plus the 5 most recent messages. Called by the
// dashboard facade.
// @Summary Messages dashboard stats (internal)
// @Description Returns the message count and most recent messages used by the dashboard facade.
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /internal/stats [get]
func (h *Handler) InternalStatsMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var messageCount int64
	h.DB.Model(&models.Message{}).Count(&messageCount)

	var recent []models.Message
	h.DB.Order("created_at DESC").Limit(5).Find(&recent)
	messages := make([]map[string]interface{}, 0, len(recent))
	for _, m := range recent {
		messages = append(messages, map[string]interface{}{
			"id":        m.ID,
			"title":     m.Title,
			"content":   m.Content,
			"createdAt": m.CreatedAt,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages":       messageCount,
		"recentMessages": messages,
	})
}

// last7DaysRegistrations returns the zero-filled daily registration counts for
// the last 7 days (oldest first), preserving the Phase 26 dashboard chart
// contract: [{date: "2006-01-02", count: N}].
func last7DaysRegistrations(db *gorm.DB) []map[string]interface{} {
	now := time.Now()
	registrations := make([]map[string]interface{}, 0, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		end := start.AddDate(0, 0, 1)
		var count int64
		db.Model(&models.User{}).
			Where("created_at >= ? AND created_at < ?", start, end).
			Count(&count)
		registrations = append(registrations, map[string]interface{}{
			"date":  day.Format("2006-01-02"),
			"count": count,
		})
	}
	return registrations
}
