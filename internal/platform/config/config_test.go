package config

import (
	"strings"
	"testing"
)

func TestConfigurationValidation(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 48))
	t.Setenv("APP_ENV", "development")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("PUBLIC_API_URL", "http://localhost:8080")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")
	if _, e := Load(); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ k, v string }{{"JWT_SECRET", "short"}, {"CORS_ALLOWED_ORIGINS", "*"}, {"DB_MAX_CONNS", "999"}, {"COOKIE_SECURE", "invalid"}, {"PUBLIC_API_URL", "javascript:foo"}, {"TRUSTED_PROXY_CIDRS", "garbage"}, {"APP_ENV", "production"}} {
		t.Run(tc.k, func(t *testing.T) {
			t.Setenv(tc.k, tc.v)
			if _, e := Load(); e == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
}
