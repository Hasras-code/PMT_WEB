package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/notification"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) adminRoutes(r chi.Router) {
	b := batch.Service{Pool: a.Pool}
	u := user.Service{Pool: a.Pool}
	r.Post("/admin/batches", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[batch.Input](w, r)
		if e != nil {
			return e
		}
		id, e := b.Create(r.Context(), userID(r), in, false)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Post("/admin/batches/{batchID}/archive", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := b.Archive(r.Context(), userID(r), batchID(r)); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	for _, path := range []string{"/admin/users", "/admin/users/{userID}"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			l, o, e := page(r)
			if e != nil {
				return e
			}
			v, e := u.AdminList(r.Context(), userID(r), param(r, "userID"), l, o)
			if e != nil {
				return e
			}
			return send(w, 200, v)
		}))
	}
	for _, x := range []struct{ path, status string }{{"suspend", "SUSPENDED"}, {"reactivate", "ACTIVE"}, {"archive", "ARCHIVED"}} {
		r.Post("/admin/users/{userID}/"+x.path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			if e := u.Status(r.Context(), userID(r), param(r, "userID"), x.status); e != nil {
				return e
			}
			return send(w, 204, nil)
		}))
	}
	r.Get("/admin/audit-logs", a.auditHandler())
}
func (a *API) auditHandler() http.HandlerFunc {
	return a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := audit.List(r.Context(), a.Pool, userID(r), batchID(r), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	})
}
func (a *API) notificationRoutes(r chi.Router) {
	s := notification.Service{Pool: a.Pool}
	r.Get("/me/notifications", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.List(r.Context(), userID(r), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/me/notifications/{notificationID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Read(r.Context(), userID(r), param(r, "notificationID")); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/me/notifications/read-all", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Read(r.Context(), userID(r), ""); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
