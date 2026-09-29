package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

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

func TestRefreshCookieSameSiteMatchesDeployment(t *testing.T) {
	for _, tc := range []struct {
		env      string
		secure   bool
		sameSite http.SameSite
	}{
		{"development", false, http.SameSiteLaxMode},
		{"production", true, http.SameSiteNoneMode},
	} {
		t.Run(tc.env, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			api := API{Config: config.Config{Env: tc.env, CookieSecure: tc.secure}}
			api.cookie(recorder, "token", time.Now().Add(time.Hour), 3600)
			response := recorder.Result()
			cookies := response.Cookies()
			if len(cookies) != 1 || cookies[0].SameSite != tc.sameSite || cookies[0].Secure != tc.secure || !cookies[0].HttpOnly {
				t.Fatalf("cookie = %+v", cookies)
			}
		})
	}
}
