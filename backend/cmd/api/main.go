package main

import (
	"log"
	"net/http"

	"omoikane-backend/internal/config"
	"omoikane-backend/internal/database"
	"omoikane-backend/internal/handlers"
)

func main() {
	cfg := config.Load()

	db := database.MustConnect(cfg.DatabaseURL)
	database.MustAutoMigrate(db)

	uploadDir := cfg.UploadDir
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	h := &handlers.Handler{DB: db, JWTSecret: cfg.JWTSecret, UploadDir: uploadDir}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Setup check (public)
	mux.HandleFunc("GET /setup/check", h.SetupStatus)

	// Auth (public)
	mux.HandleFunc("POST /setup", h.Setup)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/logout", h.Logout)
	mux.HandleFunc("POST /auth/forgot-password", h.ForgotPassword)

	// Settings
	mux.HandleFunc("GET /settings", h.GetSettings) // public (read)
	mux.HandleFunc("PUT /settings", h.Admin(h.UpdateSettings))
	mux.HandleFunc("GET /settings/profile", h.Auth(h.GetProfile))
	mux.HandleFunc("PUT /settings/profile", h.Auth(h.UpdateProfile))
	mux.HandleFunc("POST /settings/password", h.Auth(h.ChangePassword))

	// Dashboard
	mux.HandleFunc("GET /dashboard", h.Admin(h.GetDashboard))
	mux.HandleFunc("GET /dashboard/stats", h.Admin(h.GetDashboardStats))

	// Messages
	mux.HandleFunc("GET /messages", h.Auth(h.GetMessages))
	mux.HandleFunc("POST /messages", h.Admin(h.CreateMessage))
	mux.HandleFunc("GET /messages/{id}", h.Auth(h.GetMessage))
	mux.HandleFunc("POST /messages/{id}/read", h.Auth(h.MarkRead))
	mux.HandleFunc("POST /messages/read-all", h.Auth(h.MarkAllRead))
	mux.HandleFunc("DELETE /messages", h.Admin(h.DeleteAllMessages))
	mux.HandleFunc("DELETE /messages/{id}", h.Admin(h.DeleteMessage))

	// Media
	mux.HandleFunc("GET /media", h.Auth(h.GetMedia))
	mux.HandleFunc("POST /media", h.Auth(h.UploadMedia))
	mux.HandleFunc("GET /media/{id}", h.Auth(h.GetMediaItem))
	mux.HandleFunc("DELETE /media/{id}", h.Auth(h.DeleteMedia))

	// Blog
	mux.HandleFunc("GET /blog/posts", h.GetPosts)
	mux.HandleFunc("GET /admin/blog/posts", h.Admin(h.GetAdminPosts))
	mux.HandleFunc("GET /blog/posts/slug/{slug}", h.GetPostBySlug)
	mux.HandleFunc("GET /blog/posts/{id}", h.GetPost)
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	mux.HandleFunc("PUT /blog/posts/{id}", h.Auth(h.UpdatePost))
	mux.HandleFunc("DELETE /blog/posts/{id}", h.Auth(h.DeletePost))
	mux.HandleFunc("POST /blog/posts/{id}/like", h.Auth(h.ToggleLike))
	mux.HandleFunc("GET /blog/tags", h.GetTags)
	mux.HandleFunc("POST /blog/tags", h.Admin(h.CreateTag))
	mux.HandleFunc("DELETE /blog/tags/{id}", h.Admin(h.DeleteTag))
	mux.HandleFunc("GET /blog/categories", h.GetCategories)
	mux.HandleFunc("POST /blog/categories", h.Admin(h.CreateCategory))

	// Pages
	mux.HandleFunc("GET /pages", h.GetPages)
	mux.HandleFunc("GET /pages/slug/{slug}", h.GetPageBySlug)
	mux.HandleFunc("GET /pages/{id}", h.GetPage)
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("PUT /pages/{id}", h.Auth(h.UpdatePage))
	mux.HandleFunc("DELETE /pages/{id}", h.Auth(h.DeletePage))
	mux.HandleFunc("PUT /pages/reorder", h.Auth(h.ReorderPages))

	// Users (admin only)
	mux.HandleFunc("GET /users", h.Admin(h.GetUsers))
	mux.HandleFunc("POST /users", h.Admin(h.CreateUser))
	mux.HandleFunc("PUT /users/{id}", h.Admin(h.UpdateUser))
	mux.HandleFunc("DELETE /users/{id}", h.Admin(h.DeleteUser))

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
