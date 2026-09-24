// Command trash is the Omoikane trash service (Phase 31, Wave 3).
//
// It is a thin cross-cutting aggregator (service-boundaries §2): the trash
// surface spans every entity type (user/page/post/media/contact/message/tag/
// category) with one path shape, so no single owning service can prefix-split
// it. This service owns GET/POST/DELETE /trash* for the gateway and fans out
// each operation to the owning services' /internal/trash endpoints. It has NO
// database of its own.
//
// Ownership: user → auth-service, page/post/tag/category → content-service,
// media → media-service (disk cleanup happens there), contact/message →
// messages-service.
//
// Auth: the gateway protects /api/trash* as an admin surface. This service
// validates the admin JWT itself (no TokenStore/DB needed — the claims are
// self-contained), then forwards the operation with the shared internal token.
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
	"omoikane-backend/internal/observability"
)

func main() {
	// JSON structured logs + process-wide Prometheus registry (Phase 34).
	observability.Setup("trash")

	port := os.Getenv("TRASH_PORT")
	if port == "" {
		port = "8087"
	}

	cfg := trashCfg{
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		InternalToken: getEnv("INTERNAL_TOKEN", ""),
		AuthURL:       strings.TrimRight(getEnv("AUTH_SERVICE_URL", "http://auth-service:8082"), "/"),
		ContentURL:    strings.TrimRight(getEnv("CONTENT_SERVICE_URL", "http://content-service:8083"), "/"),
		MediaURL:      strings.TrimRight(getEnv("MEDIA_SERVICE_URL", "http://media-service:8084"), "/"),
		MessagesURL:   strings.TrimRight(getEnv("MESSAGES_SERVICE_URL", "http://messages-service:8085"), "/"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := newTrashMux(cfg)

	addr := ":" + port
	log.Printf("trash-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("trash-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("trash-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("trash-service: shutdown error: %v", err)
	}
}

type trashCfg struct {
	JWTSecret, InternalToken string
	AuthURL, ContentURL      string
	MediaURL, MessagesURL    string
}

// newTrashMux registers every trash-service route. Mux paths carry NO /api
// prefix — nginx prefix locations strip the prefix, so /api/trash -> /trash.
// Exposed as a function so cmd/trash tests can exercise the full wiring.
func newTrashMux(cfg trashCfg) *http.ServeMux {
	facade := &handlers.TrashFacade{
		AuthURL:       cfg.AuthURL,
		ContentURL:    cfg.ContentURL,
		MediaURL:      cfg.MediaURL,
		MessagesURL:   cfg.MessagesURL,
		InternalToken: cfg.InternalToken,
	}

	mux := http.NewServeMux()

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Prometheus scrape endpoint (Phase 34); service-level, never gateway-exposed.
	mux.Handle("GET /metrics", observability.MetricsHandler())

	// Trash surface (admin only; JWT claims are self-contained so the facade
	// needs no DB. Bearer API tokens are NOT accepted here — acceptable loss).
	mux.HandleFunc("GET /trash", middleware.AdminRequired(cfg.JWTSecret, facade.GetTrash))
	mux.HandleFunc("GET /trash/count", middleware.AdminRequired(cfg.JWTSecret, facade.GetTrashCount))
	mux.HandleFunc("POST /trash/{entity}/{id}/restore", middleware.AdminRequired(cfg.JWTSecret, facade.RestoreItem))
	mux.HandleFunc("DELETE /trash/{entity}/{id}", middleware.AdminRequired(cfg.JWTSecret, facade.HardDeleteItem))
	mux.HandleFunc("DELETE /trash", middleware.AdminRequired(cfg.JWTSecret, facade.EmptyTrash))

	return mux
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
