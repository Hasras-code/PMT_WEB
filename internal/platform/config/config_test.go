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
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 48))
	t.Setenv("APP_ENV", "production")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("PUBLIC_API_URL", "https://api.example.com")
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
