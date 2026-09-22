// Command dashboard is the Omoikane dashboard service (Phase 31, Wave 3).
//
// It is an aggregator facade (service-boundaries §3): /dashboard and
// /dashboard/stats read across users, content, media and messages by calling
// each owning service's /internal/stats endpoint. It has NO database of its
// own and reproduces the exact Phase 26 dashboard JSON contract (counts +
// zero-filled 7-day registrations + recent messages).
//
// Auth: the gateway protects /api/dashboard* as an admin surface. This service
// validates the admin JWT itself (claims are self-contained), then fetches the
// internal stats with the shared internal token.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/middleware"
)

func main() {
	port := os.Getenv("DASHBOARD_PORT")
	if port == "" {
		port = "8088"
	}

	cfg := dashboardCfg{
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		InternalToken: getEnv("INTERNAL_TOKEN", ""),
		AuthURL:       strings.TrimRight(getEnv("AUTH_SERVICE_URL", "http://auth-service:8082"), "/"),
		ContentURL:    strings.TrimRight(getEnv("CONTENT_SERVICE_URL", "http://content-service:8083"), "/"),
		MediaURL:      strings.TrimRight(getEnv("MEDIA_SERVICE_URL", "http://media-service:8084"), "/"),
		MessagesURL:   strings.TrimRight(getEnv("MESSAGES_SERVICE_URL", "http://messages-service:8085"), "/"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := newDashboardMux(cfg)

	addr := ":" + port
	log.Printf("dashboard-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("dashboard-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("dashboard-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("dashboard-service: shutdown error: %v", err)
	}
}

type dashboardCfg struct {
	JWTSecret, InternalToken string
	AuthURL, ContentURL      string
	MediaURL, MessagesURL    string
}

// newDashboardMux registers every dashboard-service route. Mux paths carry NO
// /api prefix — nginx prefix locations strip the prefix, so /api/dashboard ->
// /dashboard. Exposed as a function so cmd/dashboard tests can exercise the
// full wiring.
func newDashboardMux(cfg dashboardCfg) *http.ServeMux {
	facade := &handlers.DashboardFacade{
		AuthURL:       cfg.AuthURL,
		ContentURL:    cfg.ContentURL,
		MediaURL:      cfg.MediaURL,
		MessagesURL:   cfg.MessagesURL,
		InternalToken: cfg.InternalToken,
	}

	mux := http.NewServeMux()

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Dashboard surface (admin only; JWT claims are self-contained so the
	// facade needs no DB).
	mux.HandleFunc("GET /dashboard", middleware.AdminRequired(cfg.JWTSecret, facade.GetDashboard))
	mux.HandleFunc("GET /dashboard/stats", middleware.AdminRequired(cfg.JWTSecret, facade.GetDashboardStats))

	return mux
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
