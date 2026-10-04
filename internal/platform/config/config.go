package config

import (
	"fmt"
	"net"
	"net/mail"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	TrustedProxies  []netip.Prefix
	Env, Addr       string
	BaseURL         string
	FrontendURL     string
	DatabaseURL     string
	Secret          string
	Issuer          string
	Audience        string
	StorageDir      string
	StorageProvider string
	R2Endpoint      string
	R2Region        string
	R2AccessKeyID   string
	R2SecretKey     string
	R2PrivateBucket string
	R2PublicBucket  string
	R2PublicBaseURL string
	SMTP, MailFrom  string
	SMTPUsername    string
	SMTPPassword    string
	SMTPTLSMode     string
	BasicUser       string
	BasicPass       string
	Origins         []string
	MaxConns        int32
	CookieSecure    bool
	UploadURLTTL    time.Duration
	DownloadURLTTL  time.Duration
}

func Load() (Config, error) {
	c := Config{
		Env:             strings.ToLower(get("APP_ENV", "development")),
		Addr:            get("HTTP_ADDR", ":"+get("PORT", "8080")),
		BaseURL:         get("PUBLIC_API_URL", "http://localhost:8080"),
		FrontendURL:     strings.TrimRight(get("FRONTEND_URL", "http://localhost:5173"), "/"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		Secret:          os.Getenv("JWT_SECRET"),
		Issuer:          get("JWT_ISSUER", "pmt-api"),
		Audience:        get("JWT_AUDIENCE", "pmt-clients"),
		StorageDir:      get("STORAGE_DIR", "./data/files"),
		StorageProvider: strings.ToLower(get("STORAGE_PROVIDER", "local")),
		R2Endpoint:      strings.TrimRight(os.Getenv("R2_ENDPOINT"), "/"),
		R2Region:        get("R2_REGION", "auto"),
		R2AccessKeyID:   os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretKey:     os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2PrivateBucket: os.Getenv("R2_PRIVATE_BUCKET"),
		R2PublicBucket:  os.Getenv("R2_PUBLIC_BUCKET"),
		R2PublicBaseURL: strings.TrimRight(os.Getenv("R2_PUBLIC_BASE_URL"), "/"),
		SMTP:            get("SMTP_ADDR", "localhost:1025"),
		MailFrom:        get("MAIL_FROM", "lms@localhost"),
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SMTPTLSMode:     strings.ToLower(get("SMTP_TLS_MODE", "none")),
		BasicUser:       get("AUTH_BASIC_USER", "admin"),
		BasicPass:       get("AUTH_BASIC_PASS", "admin123"),
	}
	n, err := strconv.Atoi(get("DB_MAX_CONNS", "5"))
	if err != nil || n < 1 || n > 100 {
		return c, fmt.Errorf("invalid DB_MAX_CONNS")
	}
	c.MaxConns = int32(n)
	c.UploadURLTTL, err = seconds("UPLOAD_URL_TTL_SECONDS", 600)
	if err != nil {
		return c, err
	}
	c.DownloadURLTTL, err = seconds("DOWNLOAD_URL_TTL_SECONDS", 300)
	if err != nil {
		return c, err
	}
	c.CookieSecure, err = strconv.ParseBool(get("COOKIE_SECURE", "false"))
	if err != nil {
		return c, fmt.Errorf("invalid COOKIE_SECURE")
	}
	if c.DatabaseURL == "" || len(c.Secret) < 32 || strings.HasPrefix(c.Secret, "replace-") {
		return c, fmt.Errorf("DATABASE_URL and a non-placeholder JWT_SECRET of at least 32 bytes are required")
	}
	if c.Env != "development" && c.Env != "test" && c.Env != "production" {
		return c, fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if c.SMTPTLSMode != "none" && c.SMTPTLSMode != "starttls" && c.SMTPTLSMode != "implicit" {
		return c, fmt.Errorf("SMTP_TLS_MODE must be none, starttls, or implicit")
	}
	if _, err = mail.ParseAddress(c.MailFrom); err != nil {
		return c, fmt.Errorf("invalid MAIL_FROM")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return c, fmt.Errorf("invalid PUBLIC_API_URL")
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	frontendURL, err := url.Parse(c.FrontendURL)
	if err != nil || frontendURL.Host == "" || (frontendURL.Scheme != "http" && frontendURL.Scheme != "https") || frontendURL.Path != "" || frontendURL.RawQuery != "" || frontendURL.Fragment != "" {
		return c, fmt.Errorf("invalid FRONTEND_URL")
	}
	if c.Env == "production" && frontendURL.Scheme != "https" {
		return c, fmt.Errorf("production FRONTEND_URL must use HTTPS")
	}
	if c.StorageProvider != "local" && c.StorageProvider != "r2" {
		return c, fmt.Errorf("STORAGE_PROVIDER must be local or r2")
	}
	if c.Env == "production" && c.StorageProvider != "r2" {
		return c, fmt.Errorf("production requires STORAGE_PROVIDER=r2")
	}
	if c.StorageProvider == "r2" {
		if c.R2Endpoint == "" || c.R2AccessKeyID == "" || c.R2SecretKey == "" || c.R2PrivateBucket == "" || c.R2PublicBucket == "" || c.R2PublicBaseURL == "" {
			return c, fmt.Errorf("R2_ENDPOINT, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_PRIVATE_BUCKET, R2_PUBLIC_BUCKET, and R2_PUBLIC_BASE_URL are required for R2 storage")
		}
		for name, value := range map[string]string{
			"R2_ENDPOINT": c.R2Endpoint, "R2_ACCESS_KEY_ID": c.R2AccessKeyID,
			"R2_SECRET_ACCESS_KEY": c.R2SecretKey, "R2_PRIVATE_BUCKET": c.R2PrivateBucket,
			"R2_PUBLIC_BUCKET": c.R2PublicBucket, "R2_PUBLIC_BASE_URL": c.R2PublicBaseURL,
		} {
			if placeholder(value) {
				return c, fmt.Errorf("%s contains a placeholder", name)
			}
		}
		for name, raw := range map[string]string{"R2_ENDPOINT": c.R2Endpoint, "R2_PUBLIC_BASE_URL": c.R2PublicBaseURL} {
			v, e := url.Parse(raw)
			if e != nil || v.Scheme != "https" || v.Host == "" || v.RawQuery != "" || v.Fragment != "" {
				return c, fmt.Errorf("invalid %s", name)
			}
			if c.Env == "production" && name == "R2_PUBLIC_BASE_URL" && strings.HasSuffix(strings.ToLower(v.Hostname()), ".r2.dev") {
				return c, fmt.Errorf("production R2_PUBLIC_BASE_URL requires a custom domain")
			}
		}
	}
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
	if c.Env == "production" {
		if err = validateProductionDatabaseURL(c.DatabaseURL); err != nil {
			return c, err
		}
		if len(c.Origins) == 0 {
			return c, fmt.Errorf("production requires CORS_ALLOWED_ORIGINS")
		}
		frontendAllowed := false
		for _, origin := range c.Origins {
			frontendAllowed = frontendAllowed || origin == c.FrontendURL
		}
		if !frontendAllowed {
			return c, fmt.Errorf("FRONTEND_URL must be included in CORS_ALLOWED_ORIGINS")
		}
		basic := strings.ToLower(strings.TrimSpace(c.BasicPass))
		if strings.TrimSpace(c.BasicUser) == "" || len(c.BasicPass) < 16 || basic == "admin123" || strings.Contains(basic, "change-") || strings.Contains(basic, "replace-") {
			return c, fmt.Errorf("production requires non-placeholder Basic authentication credentials")
		}
		if c.SMTPTLSMode == "none" || c.SMTPUsername == "" || c.SMTPPassword == "" || placeholder(c.SMTPUsername) || placeholder(c.SMTPPassword) {
			return c, fmt.Errorf("production requires authenticated TLS SMTP configuration")
		}
		if host, _, splitErr := net.SplitHostPort(c.SMTP); splitErr != nil || host == "" || host == "localhost" || host == "127.0.0.1" {
			return c, fmt.Errorf("production requires a valid remote SMTP_ADDR")
		}
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

func validateProductionDatabaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" {
		return fmt.Errorf("production DATABASE_URL must be a valid PostgreSQL URL")
	}
	databaseName := strings.Trim(u.Path, "/")
	if databaseName == "" || strings.Contains(databaseName, "/") {
		return fmt.Errorf("production DATABASE_URL must include one database name")
	}
	switch strings.ToLower(u.Query().Get("sslmode")) {
	case "require", "verify-ca", "verify-full":
		return nil
	default:
		return fmt.Errorf("production DATABASE_URL must require TLS")
	}
}

func placeholder(value string) bool {
	v := strings.ToLower(value)
	return strings.Contains(v, "replace-") || strings.Contains(v, "your-") || strings.Contains(v, "<") || strings.Contains(v, ">")
}
func get(k, d string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return d
}

func seconds(key string, fallback int) (time.Duration, error) {
	n, err := strconv.Atoi(get(key, strconv.Itoa(fallback)))
	if err != nil || n < 60 || n > 3600 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return time.Duration(n) * time.Second, nil
}
