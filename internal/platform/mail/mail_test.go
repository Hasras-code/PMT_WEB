package mail

import (
	"context"
	"strings"
	"testing"
)

func TestSendRejectsUnsafeConfigurationBeforeDial(t *testing.T) {
	for _, tc := range []struct {
		name string
		mail SMTP
		to   string
	}{
		{"header injection", SMTP{Addr: "localhost:1025", From: "lms@example.com"}, "user@example.com\r\nBcc: attacker@example.com"},
		{"invalid TLS mode", SMTP{Addr: "localhost:1025", From: "lms@example.com", TLSMode: "optional"}, "user@example.com"},
		{"credentials without TLS", SMTP{Addr: "localhost:1025", From: "lms@example.com", Username: "user", Password: "secret"}, "user@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.mail.Send(context.Background(), tc.to, "TEST", strings.Repeat("t", 16))
			if err == nil {
				t.Fatal("Send accepted unsafe mail configuration")
			}
		})
	}
}

func TestMessageRendersVerificationTemplate(t *testing.T) {
	mailer := SMTP{
		Addr:        "localhost:1025",
		From:        "noreply@example.com",
		FrontendURL: "https://pmt.example.com",
	}

	raw, err := mailer.message("noreply@example.com", "student@example.com", "EMAIL_VERIFY", "verify-token-123")
	if err != nil {
		t.Fatalf("unexpected error rendering message: %v", err)
	}

	body := string(raw)
	if !strings.Contains(body, "https://pmt.example.com/verify#token=verify-token-123") {
		t.Errorf("expected verification URL in email body, got: %s", body)
	}
	if !strings.Contains(body, "verify-token-123") {
		t.Errorf("expected token in email body, got: %s", body)
	}
	if !strings.Contains(body, "Verify your PMT Family email") {
		t.Errorf("expected subject in email body, got: %s", body)
	}
}
