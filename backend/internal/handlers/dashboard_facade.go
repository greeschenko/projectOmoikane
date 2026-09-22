package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// typed internal stat payloads returned by the owning services' /internal/stats.
type authInternalStats struct {
	Users               int64                    `json:"users"`
	RecentRegistrations []map[string]interface{} `json:"recentRegistrations"`
}

type contentInternalStats struct {
	Pages int64 `json:"pages"`
	Posts int64 `json:"posts"`
}

type mediaInternalStats struct {
	Media int64 `json:"media"`
}

type messagesInternalStats struct {
	Messages       int64                    `json:"messages"`
	RecentMessages []map[string]interface{} `json:"recentMessages"`
}

// DashboardFacade is the aggregator facade (Phase 31 §3). It owns no data
// layer: it fetches each owning service's /internal/stats payload (authenticated
// with the shared internal token) and merges them into the Phase 26 dashboard
// JSON contract.
type DashboardFacade struct {
	AuthURL, ContentURL, MediaURL, MessagesURL string
	InternalToken                              string
	Client                                     *http.Client
}

func (f *DashboardFacade) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (f *DashboardFacade) fetchJSON(url string, dst interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if f.InternalToken != "" {
		req.Header.Set("X-Internal-Token", f.InternalToken)
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("internal stats %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (f *DashboardFacade) fetchAuth() (authInternalStats, error) {
	var s authInternalStats
	if f.AuthURL == "" {
		return s, nil
	}
	err := f.fetchJSON(f.AuthURL+"/internal/stats", &s)
	return s, err
}

func (f *DashboardFacade) fetchContent() (contentInternalStats, error) {
	var s contentInternalStats
	if f.ContentURL == "" {
		return s, nil
	}
	err := f.fetchJSON(f.ContentURL+"/internal/stats", &s)
	return s, err
}

func (f *DashboardFacade) fetchMedia() (mediaInternalStats, error) {
	var s mediaInternalStats
	if f.MediaURL == "" {
		return s, nil
	}
	err := f.fetchJSON(f.MediaURL+"/internal/stats", &s)
	return s, err
}

func (f *DashboardFacade) fetchMessages() (messagesInternalStats, error) {
	var s messagesInternalStats
	if f.MessagesURL == "" {
		return s, nil
	}
	err := f.fetchJSON(f.MessagesURL+"/internal/stats", &s)
	return s, err
}

func (f *DashboardFacade) writeServiceError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]string{"error": "Dashboard service unavailable"})
}

// GetDashboard returns the aggregated content counts (admin only).
// @Summary Dashboard content counts
// @Description Returns counts of users, pages, posts, media and messages.
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard [get]
func (f *DashboardFacade) GetDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	auth, err := f.fetchAuth()
	if err != nil {
		f.writeServiceError(w)
		return
	}
	content, err := f.fetchContent()
	if err != nil {
		f.writeServiceError(w)
		return
	}
	media, err := f.fetchMedia()
	if err != nil {
		f.writeServiceError(w)
		return
	}
	messages, err := f.fetchMessages()
	if err != nil {
		f.writeServiceError(w)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"users":    auth.Users,
		"pages":    content.Pages,
		"posts":    content.Posts,
		"media":    media.Media,
		"messages": messages.Messages,
	})
}

// GetDashboardStats returns the aggregated dashboard stats (admin only).
// @Summary Dashboard stats
// @Description Returns content counts plus the 5 most recent user registrations and messages.
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/stats [get]
func (f *DashboardFacade) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	auth, err := f.fetchAuth()
	if err != nil {
		f.writeServiceError(w)
		return
	}
	content, err := f.fetchContent()
	if err != nil {
		f.writeServiceError(w)
		return
	}
	media, err := f.fetchMedia()
	if err != nil {
		f.writeServiceError(w)
		return
	}
	messages, err := f.fetchMessages()
	if err != nil {
		f.writeServiceError(w)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"userCount":           auth.Users,
		"pageCount":           content.Pages,
		"blogCount":           content.Posts,
		"mediaCount":          media.Media,
		"recentMessages":      messages.RecentMessages,
		"recentRegistrations": auth.RecentRegistrations,
	})
}
