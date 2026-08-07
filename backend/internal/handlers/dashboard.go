package handlers

import (
	"encoding/json"
	"net/http"

	"omoikane-backend/internal/models"
)

// GetDashboard returns content counts (admin only).
// @Summary Dashboard content counts
// @Description Returns counts of users, pages, posts, media and messages.
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard [get]
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var userCount int64
	var pageCount int64
	var postCount int64
	var mediaCount int64
	var messageCount int64

	h.DB.Model(&models.User{}).Count(&userCount)
	h.DB.Model(&models.Page{}).Count(&pageCount)
	h.DB.Model(&models.BlogPost{}).Count(&postCount)
	h.DB.Model(&models.MediaItem{}).Count(&mediaCount)
	h.DB.Model(&models.Message{}).Count(&messageCount)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"users":    userCount,
		"pages":    pageCount,
		"posts":    postCount,
		"media":    mediaCount,
		"messages": messageCount,
	})
}

// GetDashboardStats returns dashboard stats plus recent registrations and messages (admin only).
// @Summary Dashboard stats
// @Description Returns content counts plus the 5 most recent user registrations and messages.
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/stats [get]
func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var userCount int64
	var pageCount int64
	var postCount int64
	var mediaCount int64

	h.DB.Model(&models.User{}).Count(&userCount)
	h.DB.Model(&models.Page{}).Count(&pageCount)
	h.DB.Model(&models.BlogPost{}).Count(&postCount)
	h.DB.Model(&models.MediaItem{}).Count(&mediaCount)

	var recentUsers []models.User
	h.DB.Order("created_at DESC").Limit(5).Find(&recentUsers)
	registrations := make([]map[string]interface{}, 0, len(recentUsers))
	for _, u := range recentUsers {
		registrations = append(registrations, map[string]interface{}{
			"id":   u.ID,
			"name": u.Name,
			"email": u.Email,
			"role": u.Role,
			"createdAt": u.CreatedAt,
		})
	}

	var recentMessages []models.Message
	h.DB.Order("created_at DESC").Limit(5).Find(&recentMessages)
	messages := make([]map[string]interface{}, 0, len(recentMessages))
	for _, m := range recentMessages {
		messages = append(messages, map[string]interface{}{
			"id":        m.ID,
			"title":     m.Title,
			"content":   m.Content,
			"createdAt": m.CreatedAt,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"userCount":          userCount,
		"pageCount":          pageCount,
		"blogCount":          postCount,
		"mediaCount":         mediaCount,
		"recentMessages":     messages,
		"recentRegistrations": registrations,
	})
}
