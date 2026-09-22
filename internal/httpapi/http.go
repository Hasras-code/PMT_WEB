package httpapi

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	Pool               *pgxpool.Pool
	Auth               *auth.Service
	Files              *storage.Local
	Uploads            upload.Service
	Config             config.Config
	Log                *slog.Logger
	swaggerOnce        sync.Once
	swaggerDocument    []byte
	swaggerDocumentErr error
	swaggerRouter      chi.Router
}
type endpoint func(http.ResponseWriter, *http.Request) error
type contextKey int

const (
	identityKey contextKey = iota
	requestIDKey
)

func identity(r *http.Request) auth.Claims {
	c, _ := r.Context().Value(identityKey).(auth.Claims)
	return c
}
func userID(r *http.Request) string          { return identity(r).Subject }
func param(r *http.Request, k string) string { return chi.URLParam(r, k) }
func batchID(r *http.Request) string         { return param(r, "batchID") }
func send(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status == 204 {
		return nil
	}
	return json.NewEncoder(w).Encode(v)
}
func created(w http.ResponseWriter, id string) error {
	return send(w, 201, map[string]string{"id": id})
}
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		return v, apperror.ErrInvalid
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		return v, e
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return v, errJSON
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&v); e != nil {
		return v, errJSON
	}
	var extra any
	if e := d.Decode(&extra); !errors.Is(e, io.EOF) {
		return v, errJSON
	}
	return v, nil
}

var errJSON = errors.New("invalid JSON body")

func page(r *http.Request) (int, int, error) {
	limit, offset := 20, 0
	var e error
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, e = strconv.Atoi(v)
		if e != nil || limit < 1 || limit > 100 {
			return 0, 0, apperror.ErrInvalid
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		offset, e = strconv.Atoi(v)
		if e != nil || offset < 0 || offset > 100000 {
			return 0, 0, apperror.ErrInvalid
		}
	}
	return limit, offset, nil
}
func (a *API) wrap(h endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for i, k := range chi.RouteContext(r.Context()).URLParams.Keys {
			if strings.HasSuffix(k, "ID") {
				if _, e := uuid.Parse(chi.RouteContext(r.Context()).URLParams.Values[i]); e != nil {
					a.fail(w, r, apperror.ErrInvalid)
					return
				}
			}
		}
		if e := h(w, r); e != nil {
			a.fail(w, r, e)
		}
	}
}
func (a *API) fail(w http.ResponseWriter, r *http.Request, e error) {
	e = db.Error(e)
	status, code, message := 500, "internal_error", "An internal error occurred"
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(e, &tooLarge):
		status, code, message = 413, "body_too_large", "Request body is too large"
	case errors.Is(e, errJSON):
		status, code, message = 400, "invalid_json", "Request body must contain one valid JSON object with known fields"
	case errors.Is(e, apperror.ErrInvalid):
		status, code, message = 422, "validation_failed", "Request validation failed"
	case errors.Is(e, apperror.ErrNotFound):
		status, code, message = 404, "not_found", "Not found"
	case errors.Is(e, apperror.ErrUnauthorized):
		status, code, message = 401, "unauthorized", "Invalid credentials"
	case errors.Is(e, apperror.ErrForbidden):
		status, code, message = 403, "forbidden", "You are not allowed to perform this action"
	case errors.Is(e, apperror.ErrConflict):
		status, code, message = 409, "conflict", "Conflict or invalid state"
	}
	if domainCode := apperror.Code(e); domainCode != "" {
		code = domainCode
	}
	if status == 500 {
		a.Log.Error("request failed", "request_id", r.Context().Value(requestIDKey))
	}
	_ = send(w, status, map[string]any{"error": map[string]any{"code": code, "message": message, "request_id": r.Context().Value(requestIDKey)}})
}
func (a *API) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			a.fail(w, r, apperror.ErrUnauthorized)
			return
		}
		c, e := a.Auth.Authenticate(r.Context(), strings.TrimPrefix(header, "Bearer "))
		if e != nil {
			a.fail(w, r, e)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey, c)))
	})
}
func (a *API) authenticatedBasic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(username), []byte(a.Config.BasicUser)) != 1 || subtle.ConstantTimeCompare([]byte(password), []byte(a.Config.BasicPass)) != 1 {
			// A Basic challenge on a cross-origin XHR makes browsers display their
			// native sign-in dialog. The frontend already knows the auth scheme and
			// supplies the header explicitly, so only challenge non-browser clients.
			if r.Header.Get("Origin") == "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			}
			a.fail(w, r, apperror.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *API) origin(origin string) bool {
	if origin == a.Config.BaseURL {
		return true
	}
	for _, v := range a.Config.Origins {
		if origin == v {
			return true
		}
	}
	return false
}
func (a *API) middleware(next http.Handler) http.Handler {
	lim := limiter{entries: map[string]bucket{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey, id))
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if a.Config.Env == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		defer func() {
			if recover() != nil {
				a.fail(w, r, errors.New("panic"))
			}
		}()
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Add("Vary", "Origin")
			if !a.origin(origin) {
				a.fail(w, r, apperror.ErrForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.WriteHeader(204)
			return
		}
		ip := clientIP(r, a.Config.TrustedProxies)
		group := "general"
		max := 180
		if strings.HasPrefix(r.URL.Path, "/v1/auth/") {
			group = "auth"
			max = 30
		}
		if strings.Contains(r.URL.Path, "/uploads") {
			group = "upload"
			max = 30
		}
		if r.Method == "POST" && (strings.Contains(r.URL.Path, "/complaints") || strings.Contains(r.URL.Path, "/feedback")) {
			group = "support"
			max = 20
		}
		if !lim.allow(ip+":"+group, max) {
			w.Header().Set("Retry-After", "60")
			_ = send(w, 429, map[string]any{"error": map[string]any{"code": "rate_limited", "message": "Too many requests", "request_id": id}})
			return
		}
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		// No request bodies, query strings, capability URLs, user IDs or anonymous submission correlations.
		if group != "support" {
			route := chi.RouteContext(r.Context()).RoutePattern()
			a.Log.Info("request", "request_id", id, "method", r.Method, "route", route, "status", ww.Status(), "duration_ms", time.Since(start).Milliseconds())
		}
	})
}

type bucket struct {
	Start time.Time
	Count int
}
type limiter struct {
	mu      sync.Mutex
	entries map[string]bucket
}

func (l *limiter) allow(key string, max int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.entries) >= 10000 {
		for k, v := range l.entries {
			if now.Sub(v.Start) >= time.Minute {
				delete(l.entries, k)
			}
		}
	}
	v, ok := l.entries[key]
	if !ok && len(l.entries) >= 10000 {
		return false
	}
	if !ok || now.Sub(v.Start) >= time.Minute {
		v = bucket{Start: now}
	}
	v.Count++
	l.entries[key] = v
	return v.Count <= max
}

func clientIP(r *http.Request, trusted []netip.Prefix) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remote, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	isTrusted := func(ip netip.Addr) bool {
		for _, p := range trusted {
			if p.Contains(ip) {
				return true
			}
		}
		return false
	}
	if !isTrusted(remote) {
		return remote.String()
	}
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	if len(parts) > 20 {
		return remote.String()
	}
	for i := len(parts) - 1; i >= 0; i-- {
		ip, e := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if e != nil {
			return remote.String()
		}
		if !isTrusted(ip) || i == 0 {
			return ip.String()
		}
	}
	return remote.String()
}
