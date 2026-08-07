package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthHandler reports service health (public).
// @Summary Health check
// @Description Returns the service status.
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
