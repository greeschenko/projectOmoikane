package config

import "os"

type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	UploadDir       string
	SMTP            SMTPConfig
	RecaptchaSecret string
	AuditServiceURL string
	RedisURL        string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		SMTP: SMTPConfig{
			Host: getEnv("SMTP_HOST", ""),
			Port: getEnv("SMTP_PORT", "587"),
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("SMTP_FROM", "noreply@omoikane.local"),
		},
		RecaptchaSecret: getEnv("RECAPTCHA_SECRET", ""),
		AuditServiceURL: getEnv("AUDIT_SERVICE_URL", ""),
		RedisURL:        getEnv("REDIS_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
