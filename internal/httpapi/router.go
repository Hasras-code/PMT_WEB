package httpapi

import (
	"context"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/position"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

func (a *API) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(a.middleware)
	if a.Config.Env != "production" {
		a.swaggerRoutes(r)
	}
	r.NotFound(a.wrap(func(w http.ResponseWriter, r *http.Request) error { return apperror.ErrNotFound }))
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		_ = send(w, 405, map[string]any{"error": map[string]any{"code": "method_not_allowed", "message": "Method not allowed", "request_id": r.Context().Value(requestIDKey)}})
	})
	r.Get("/health/live", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		return send(w, 200, map[string]string{"status": "ok"})
	}))
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if e := a.Pool.Ping(ctx); e != nil {
			_ = send(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		_ = send(w, 200, map[string]string{"status": "ok"})
	})
	r.Route("/v1", func(r chi.Router) {
		a.authRoutes(r)
		a.fileRoutes(r)
		a.galleryPublicRoutes(r)
		a.publicBatchRoutes(r)
		a.publicImageRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(a.authenticated)
			a.meRoutes(r)
			a.profileImageRoutes(r)
			a.notificationRoutes(r)
			a.adminRoutes(r)
			a.galleryAdminRoutes(r)
			r.Get("/batches", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
				l, o, e := page(r)
				if e != nil {
					return e
				}
				v, e := (batch.Service{Pool: a.Pool}).List(r.Context(), userID(r), l, o)
				if e != nil {
					return e
				}
				return send(w, 200, v)
			}))
			r.Route("/batches/{batchID}", func(r chi.Router) {
				a.batchRoutes(r)
				a.semesterRoutes(r)
				a.moduleRoutes(r)
				a.announcementRoutes(r)
				a.resourceRoutes(r)
				a.lessonRoutes(r)
				a.linkRoutes(r)
				a.eventRoutes(r)
				a.eventImageRoutes(r)
				a.positionRoutes(r)
				a.supportRoutes(r)
				r.Get("/audit-logs", a.auditHandler())
			})
		})
	})
	return r
}
func (a *API) publicBatchRoutes(r chi.Router) {
	b := batch.Service{Pool: a.Pool}
	p := position.Service{Pool: a.Pool}
	r.Get("/public/batches/{slug}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := b.Public(r.Context(), param(r, "slug"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	for _, path := range []string{"/public/batches/{slug}/events", "/public/batches/{slug}/events/{eventID}"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			l, o, e := page(r)
			if e != nil {
				return e
			}
			v, e := b.PublicEvents(r.Context(), param(r, "slug"), param(r, "eventID"), l, o)
			if e != nil {
				return e
			}
			return send(w, 200, v)
		}))
	}
	r.Get("/public/batches/{slug}/positions", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := p.Public(r.Context(), param(r, "slug"), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
}
