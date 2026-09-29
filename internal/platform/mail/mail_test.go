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
