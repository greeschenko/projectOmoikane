package handlers

import (
	"encoding/json"
	"net/http"

	"omoikane-backend/internal/models"
)

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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"userCount":    userCount,
		"pageCount":    pageCount,
		"blogCount":    postCount,
		"mediaCount":   mediaCount,
		"recentMessages": []interface{}{},
		"recentRegistrations": []interface{}{},
	})
}
