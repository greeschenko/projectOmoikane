package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
)

type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

type ResetEmailData struct {
	ResetLink   string
	SiteName    string
	ExpiryHours int
}

func SendEmail(cfg Config, to, subject, bodyHTML string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s", cfg.From, to, subject, bodyHTML)

	if cfg.Host == "" || cfg.User == "" {
		log.Printf("[mailer] SMTP not configured — would send email to %s with subject %q", to, subject)
		return nil
	}

	addr := cfg.Host + ":" + cfg.Port
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg))
}

func RenderResetTemplate(templateStr string, data ResetEmailData) (string, error) {
	tmpl, err := template.New("reset").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

func SendResetEmail(cfg Config, to, token, frontendURL string) error {
	resetLink := frontendURL + "/reset-password?token=" + token
	subject := "Password Reset Request"
	body := fmt.Sprintf(`<!DOCTYPE html><html><body>
<h2>Password Reset</h2>
<p>Click the link below to reset your password. This link expires in 1 hour.</p>
<p><a href="%s">%s</a></p>
<p>If you did not request this, please ignore this email.</p>
</body></html>`, resetLink, resetLink)

	return SendEmail(cfg, to, subject, body)
}
