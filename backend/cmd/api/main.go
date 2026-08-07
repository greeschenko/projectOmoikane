package main

import (
	"log"
	"net/http"
	"time"

	_ "omoikane-backend/docs"
	"omoikane-backend/internal/config"
	"omoikane-backend/internal/database"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/middleware"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Omoikane API
// @version 1.0
// @description REST API for the Omoikane CMS. Public endpoints do not require authentication; protected endpoints accept a Bearer token (or the session cookie set by login). Admin-only endpoints require an admin role.
// @contact.name Omoikane
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	db := database.MustConnect(cfg.DatabaseURL)
	database.MustAutoMigrate(db)

	uploadDir := cfg.UploadDir
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	h := &handlers.Handler{
		DB:              db,
		JWTSecret:       cfg.JWTSecret,
		UploadDir:       uploadDir,
		SMTPHost:        cfg.SMTP.Host,
		SMTPPort:        cfg.SMTP.Port,
		SMTPUser:        cfg.SMTP.User,
		SMTPPass:        cfg.SMTP.Pass,
		SMTPFrom:        cfg.SMTP.From,
		RecaptchaSecret: cfg.RecaptchaSecret,
		AuditServiceURL: cfg.AuditServiceURL,
	}

	mux := http.NewServeMux()

	// Swagger UI (public)
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler())

	// Health
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Setup check (public)
	mux.HandleFunc("GET /setup/check", h.SetupStatus)

	// Auth (public)
	mux.HandleFunc("POST /setup", h.Setup)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/logout", h.Logout)
	forgotPasswordLimiter := middleware.NewRateLimiter(1.0/300.0, 3, 15*time.Minute)
	mux.HandleFunc("POST /auth/forgot-password", forgotPasswordLimiter.Middleware(h.ForgotPassword))
	mux.HandleFunc("POST /auth/reset-password", h.ResetPassword)

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
	mux.HandleFunc("POST /media/batch", h.Auth(h.BatchMedia))

	// Blog
	mux.HandleFunc("GET /blog/posts", h.GetPosts)
	mux.HandleFunc("GET /admin/blog/posts", h.Admin(h.GetAdminPosts))
	mux.HandleFunc("GET /blog/posts/slug/{slug}", h.GetPostBySlug)
	mux.HandleFunc("GET /blog/posts/{id}", h.GetPost)
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	mux.HandleFunc("PUT /blog/posts/{id}", h.Auth(h.UpdatePost))
	mux.HandleFunc("DELETE /blog/posts/{id}", h.Auth(h.DeletePost))
	mux.HandleFunc("POST /blog/posts/batch", h.Auth(h.BatchPosts))
	mux.HandleFunc("POST /blog/posts/{id}/like", h.Auth(h.ToggleLike))
	mux.HandleFunc("GET /blog/tags", h.GetTags)
	mux.HandleFunc("POST /blog/tags", h.Admin(h.CreateTag))
	mux.HandleFunc("DELETE /blog/tags/{id}", h.Admin(h.DeleteTag))
	mux.HandleFunc("GET /blog/categories", h.GetCategories)
	mux.HandleFunc("POST /blog/categories", h.Admin(h.CreateCategory))
	mux.HandleFunc("DELETE /blog/categories/{id}", h.Admin(h.DeleteCategory))

	// Pages
	mux.HandleFunc("GET /pages", h.GetPages)
	mux.HandleFunc("GET /pages/slug/{slug}", h.GetPageBySlug)
	mux.HandleFunc("GET /pages/{id}", h.GetPage)
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("PUT /pages/{id}", h.Auth(h.UpdatePage))
	mux.HandleFunc("DELETE /pages/{id}", h.Auth(h.DeletePage))
	mux.HandleFunc("POST /pages/batch", h.Auth(h.BatchPages))
	mux.HandleFunc("PUT /pages/reorder", h.Auth(h.ReorderPages))

	// Contact Form
	mux.HandleFunc("POST /contact", h.SubmitContact)                             // public
	mux.HandleFunc("GET /contacts", h.Admin(h.GetContacts))                      // admin
	mux.HandleFunc("GET /contacts/{id}", h.Admin(h.GetContact))                  // admin
	mux.HandleFunc("POST /contacts/{id}/read", h.Admin(h.MarkContactRead))       // admin
	mux.HandleFunc("DELETE /contacts/{id}", h.Admin(h.DeleteContact))            // admin

	// Users (admin only)
	mux.HandleFunc("GET /users", h.Admin(h.GetUsers))
	mux.HandleFunc("POST /users", h.Admin(h.CreateUser))
	mux.HandleFunc("PUT /users/{id}", h.Admin(h.UpdateUser))
	mux.HandleFunc("DELETE /users/{id}", h.Admin(h.DeleteUser))
	mux.HandleFunc("POST /users/batch", h.Admin(h.BatchUsers))

	// Audit logs (admin only)
	mux.HandleFunc("GET /audit-logs", h.Admin(h.GetAuditLogs))

	// Trash system (admin only)
	mux.HandleFunc("GET /trash", h.Admin(h.GetTrash))
	mux.HandleFunc("GET /trash/count", h.Admin(h.GetTrashCount))
	mux.HandleFunc("POST /trash/{entity}/{id}/restore", h.Admin(h.RestoreItem))
	mux.HandleFunc("DELETE /trash/{entity}/{id}", h.Admin(h.HardDeleteItem))
	mux.HandleFunc("DELETE /trash", h.Admin(h.EmptyTrash))

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
