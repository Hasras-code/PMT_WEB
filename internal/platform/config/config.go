package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TrustedProxies                                                                        []netip.Prefix
	Env, Addr, BaseURL, DatabaseURL, Secret, Issuer, Audience, StorageDir, SMTP, MailFrom string
	BasicUser, BasicPass                                                                  string
	Origins                                                                               []string
	MaxConns                                                                              int32
	CookieSecure                                                                          bool
}

func Load() (Config, error) {
	c := Config{Env: get("APP_ENV", "development"), Addr: get("HTTP_ADDR", ":"+get("PORT", "8080")), BaseURL: get("PUBLIC_API_URL", "http://localhost:8080"), DatabaseURL: os.Getenv("DATABASE_URL"), Secret: os.Getenv("JWT_SECRET"), Issuer: get("JWT_ISSUER", "pmt-api"), Audience: get("JWT_AUDIENCE", "pmt-clients"), StorageDir: get("STORAGE_DIR", "./data/files"), SMTP: get("SMTP_ADDR", "localhost:1025"), MailFrom: get("MAIL_FROM", "lms@localhost"), BasicUser: get("AUTH_BASIC_USER", "admin"), BasicPass: get("AUTH_BASIC_PASS", "admin123")}
	n, err := strconv.Atoi(get("DB_MAX_CONNS", "5"))
	if err != nil || n < 1 || n > 100 {
		return c, fmt.Errorf("invalid DB_MAX_CONNS")
	}
	c.MaxConns = int32(n)
	c.CookieSecure, err = strconv.ParseBool(get("COOKIE_SECURE", "false"))
	if err != nil {
		return c, fmt.Errorf("invalid COOKIE_SECURE")
	}
	if c.DatabaseURL == "" || len(c.Secret) < 32 || strings.HasPrefix(c.Secret, "replace-") {
		return c, fmt.Errorf("DATABASE_URL and a non-placeholder JWT_SECRET of at least 32 bytes are required")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return c, fmt.Errorf("invalid PUBLIC_API_URL")
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	for _, s := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		v, e := url.Parse(s)
		if e != nil || v.Host == "" || (v.Scheme != "http" && v.Scheme != "https") || v.Path != "" || v.RawQuery != "" || v.Fragment != "" {
			return c, fmt.Errorf("invalid CORS origin")
		}
		c.Origins = append(c.Origins, s)
	}
	if c.Env == "production" && (!c.CookieSecure || u.Scheme != "https") {
		return c, fmt.Errorf("production requires HTTPS and secure cookies")
	}
	for _, raw := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, e := netip.ParsePrefix(raw)
		if e != nil {
			return c, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS")
		}
		c.TrustedProxies = append(c.TrustedProxies, prefix)
	}
	return c, nil
}
func get(k, d string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return d
}
