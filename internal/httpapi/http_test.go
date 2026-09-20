package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
)

func TestJSONValidation(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{"unknown":1}`, `{"title":"a"} {"title":"b"}`, `{`, strings.Repeat("a", (1<<20)+1)} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		_, e := decode[struct {
			Title string `json:"title"`
		}](httptest.NewRecorder(), r)
		if e == nil {
			t.Fatalf("accepted invalid body")
		}
	}
}
func TestPublicMiddleware(t *testing.T) {
	a := &API{Config: config.Config{BaseURL: "http://localhost:8080", Origins: []string{"http://localhost:3000"}, BasicUser: "admin", BasicPass: "secret"}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	router := a.Router()
	for _, x := range []struct {
		path, origin string
		status       int
	}{{"/health/live", "", 401}, {"/health/live", "https://evil.example", 403}, {"/v1/batches/00000000-0000-0000-0000-000000000000/gallery", "", 401}, {"/missing", "", 404}} {
		r := httptest.NewRequest("GET", x.path, nil)
		r.Header.Set("Origin", x.origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != x.status {
			t.Errorf("%s: %d %s", x.path, w.Code, w.Body.String())
		}
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("missing security header")
		}
	}
	r := httptest.NewRequest("GET", "/health/live", nil)
	r.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("valid health credentials: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("successful health request returned an auth challenge")
	}
	preflightRequest := httptest.NewRequest("OPTIONS", "/v1/me", nil)
	preflightRequest.Header.Set("Origin", "http://localhost:3000")
	preflightResponse := httptest.NewRecorder()
	router.ServeHTTP(preflightResponse, preflightRequest)
	if preflightResponse.Code != 204 || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("CORS preflight")
	}
}
func TestLimiterBounded(t *testing.T) {
	l := limiter{entries: map[string]bucket{}}
	if !l.allow("ip", 1) || l.allow("ip", 1) {
		t.Fatal("rate limit")
	}
}

var _ http.Handler

func TestProxyTrust(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.1:123"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 192.0.2.2")
	if got := clientIP(r, nil); got != "192.0.2.1" {
		t.Fatal("trusted unconfigured proxy")
	}
	trusted := []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}
	if got := clientIP(r, trusted); got != "203.0.113.5" {
		t.Fatal(got)
	}
	r.Header.Set("X-Forwarded-For", "garbage")
	if got := clientIP(r, trusted); got != "192.0.2.1" {
		t.Fatal("malformed chain accepted")
	}
}
