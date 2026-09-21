package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
)

func testApp(env string) *app {
	return &app{
		cfg: config.Config{
			Env:       env,
			BaseURL:   "http://localhost:8080",
			Origins:   []string{"http://localhost:3000"},
			BasicUser: "admin",
			BasicPass: "secret",
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestRouteMiddlewareBoundaries(t *testing.T) {
	router := testApp("development").mount()
	for _, tt := range []struct {
		path   string
		origin string
		status int
	}{
		{path: "/health/live", status: http.StatusUnauthorized},
		{path: "/health/live", origin: "https://evil.example", status: http.StatusForbidden},
		{path: "/v1/batches/00000000-0000-0000-0000-000000000000/gallery", status: http.StatusUnauthorized},
		{path: "/missing", status: http.StatusNotFound},
	} {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		req.Header.Set("Origin", tt.origin)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != tt.status {
			t.Errorf("%s: got %d, want %d: %s", tt.path, response.Code, tt.status, response.Body.String())
		}
		if response.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("missing security header")
		}
	}

	healthRequest := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	healthRequest.SetBasicAuth("admin", "secret")
	healthResponse := httptest.NewRecorder()
	router.ServeHTTP(healthResponse, healthRequest)
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("valid health credentials: %d %s", healthResponse.Code, healthResponse.Body.String())
	}
	if healthResponse.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("successful health request returned an auth challenge")
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, "/v1/me", nil)
	preflightRequest.Header.Set("Origin", "http://localhost:3000")
	preflightResponse := httptest.NewRecorder()
	router.ServeHTTP(preflightResponse, preflightRequest)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("CORS preflight")
	}
}

func TestSwaggerFollowsMountedRoutes(t *testing.T) {
	router := testApp("development").mount()

	redirect := httptest.NewRecorder()
	router.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/swagger", nil))
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "/swagger/index.html" {
		t.Fatalf("Swagger redirect: %d %q", redirect.Code, redirect.Header().Get("Location"))
	}

	ui := httptest.NewRecorder()
	router.ServeHTTP(ui, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if ui.Code != http.StatusOK || !strings.Contains(ui.Body.String(), "Swagger UI") {
		t.Fatalf("Swagger UI: %d", ui.Code)
	}

	specResponse := httptest.NewRecorder()
	router.ServeHTTP(specResponse, httptest.NewRequest(http.MethodGet, "/swagger/openapi.json", nil))
	if specResponse.Code != http.StatusOK {
		t.Fatalf("Swagger document: %d %s", specResponse.Code, specResponse.Body.String())
	}
	var document struct {
		OpenAPI string                     `json:"openapi"`
		Paths   map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(specResponse.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.OpenAPI != "3.0.3" || document.Paths["/v1/auth/login"] == nil {
		t.Fatal("Swagger document does not describe the mounted API")
	}
	for path := range document.Paths {
		if strings.HasPrefix(path, "/swagger") {
			t.Fatalf("Swagger documented its own asset route %q", path)
		}
	}

	production := testApp("production").mount()
	missing := httptest.NewRecorder()
	production.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("production Swagger status: got %d, want 404", missing.Code)
	}
}
