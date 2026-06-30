package mailer

import (
	"strings"
	"testing"
)

func TestSendEmail_NoSMTP(t *testing.T) {
	cfg := Config{}
	err := SendEmail(cfg, "test@test.com", "Subject", "<p>Body</p>")
	if err != nil {
		t.Errorf("Expected no error without SMTP, got %v", err)
	}
}

func TestRenderResetTemplate(t *testing.T) {
	templateStr := `<h2>Reset your password</h2>
<p>Hi {{.SiteName}} user,</p>
<p>Click <a href="{{.ResetLink}}">here</a> to reset. Expires in {{.ExpiryHours}} hour(s).</p>`

	data := ResetEmailData{
		ResetLink:   "http://localhost:3000/reset-password?token=abc123",
		SiteName:    "Test Site",
		ExpiryHours: 1,
	}

	result, err := RenderResetTemplate(templateStr, data)
	if err != nil {
		t.Fatalf("RenderResetTemplate failed: %v", err)
	}

	if !strings.Contains(result, "http://localhost:3000/reset-password?token=abc123") {
		t.Error("Expected reset link in rendered output")
	}
	if !strings.Contains(result, "Test Site") {
		t.Error("Expected site name in rendered output")
	}
	if !strings.Contains(result, "1 hour") {
		t.Error("Expected expiry hours in rendered output")
	}
}
