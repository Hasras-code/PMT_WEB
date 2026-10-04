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
	t.Setenv("STORAGE_PROVIDER", "local")
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

func TestR2Configuration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/postgres?sslmode=require")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 48))
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("PUBLIC_API_URL", "https://api.example.com")
	t.Setenv("FRONTEND_URL", "https://app.example.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("AUTH_BASIC_USER", "health")
	t.Setenv("AUTH_BASIC_PASS", strings.Repeat("b", 24))
	t.Setenv("SMTP_ADDR", "smtp.example.com:587")
	t.Setenv("MAIL_FROM", "lms@example.com")
	t.Setenv("SMTP_TLS_MODE", "starttls")
	t.Setenv("SMTP_USERNAME", "mailer")
	t.Setenv("SMTP_PASSWORD", strings.Repeat("m", 24))
	t.Setenv("STORAGE_PROVIDER", "r2")
	t.Setenv("R2_ENDPOINT", "https://account.r2.cloudflarestorage.com")
	t.Setenv("R2_REGION", "auto")
	t.Setenv("R2_ACCESS_KEY_ID", "access")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_PRIVATE_BUCKET", "private")
	t.Setenv("R2_PUBLIC_BUCKET", "public")
	t.Setenv("R2_PUBLIC_BASE_URL", "https://media.example.com/")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.R2PublicBaseURL != "https://media.example.com" {
		t.Fatalf("public base URL = %q", c.R2PublicBaseURL)
	}

	t.Setenv("R2_SECRET_ACCESS_KEY", "")
	if _, err = Load(); err == nil {
		t.Fatal("accepted incomplete R2 configuration")
	}
}

func TestProductionConfigurationRejectsUnsafeDependencies(t *testing.T) {
	setValidProductionEnvironment(t)
	if _, err := Load(); err != nil {
		t.Fatalf("valid production configuration: %v", err)
	}

	for _, tc := range []struct {
		name, key, value string
	}{
		{"database without TLS", "DATABASE_URL", "postgres://db.example.com/postgres"},
		{"missing database name", "DATABASE_URL", "postgres://db.example.com/?sslmode=require"},
		{"default health password", "AUTH_BASIC_PASS", "admin123"},
		{"plaintext SMTP", "SMTP_TLS_MODE", "none"},
		{"missing SMTP password", "SMTP_PASSWORD", ""},
		{"development R2 public URL", "R2_PUBLIC_BASE_URL", "https://bucket-id.r2.dev"},
		{"invalid environment", "APP_ENV", "prod"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("accepted unsafe production configuration")
			}
		})
	}
}

func setValidProductionEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://db.example.com/pmtdb?sslmode=require")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 48))
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("PUBLIC_API_URL", "https://api.example.com")
	t.Setenv("FRONTEND_URL", "https://app.example.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("AUTH_BASIC_USER", "health")
	t.Setenv("AUTH_BASIC_PASS", strings.Repeat("b", 24))
	t.Setenv("SMTP_ADDR", "smtp.example.com:587")
	t.Setenv("MAIL_FROM", "lms@example.com")
	t.Setenv("SMTP_TLS_MODE", "starttls")
	t.Setenv("SMTP_USERNAME", "mailer")
	t.Setenv("SMTP_PASSWORD", strings.Repeat("m", 24))
	t.Setenv("STORAGE_PROVIDER", "r2")
	t.Setenv("R2_ENDPOINT", "https://account.r2.cloudflarestorage.com")
	t.Setenv("R2_REGION", "auto")
	t.Setenv("R2_ACCESS_KEY_ID", "access")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_PRIVATE_BUCKET", "private")
	t.Setenv("R2_PUBLIC_BUCKET", "public")
	t.Setenv("R2_PUBLIC_BASE_URL", "https://media.example.com")
}
