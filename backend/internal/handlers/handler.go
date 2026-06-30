package handlers

import (
	"net/http"

	"omoikane-backend/internal/middleware"

	"gorm.io/gorm"
)

type Handler struct {
	DB              *gorm.DB
	JWTSecret       string
	UploadDir       string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	RecaptchaSecret string
}

func (h *Handler) Auth(next http.HandlerFunc) http.HandlerFunc {
	return middleware.AuthRequired(h.JWTSecret, next)
}

func (h *Handler) Admin(next http.HandlerFunc) http.HandlerFunc {
	return middleware.AdminRequired(h.JWTSecret, next)
}
