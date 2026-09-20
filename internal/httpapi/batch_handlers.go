package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/membership"

	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) batchRoutes(r chi.Router) {
	s := batch.Service{Pool: a.Pool}
	m := membership.Service{Pool: a.Pool}
	r.Get("/", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), in.Name, in.Description); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/profile", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Profile(r.Context(), userID(r), batchID(r))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/profile", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[batch.Profile](w, r)
		if e != nil {
			return e
		}
		if e = s.SetProfile(r.Context(), userID(r), batchID(r), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/profile/uploads", a.uploadHandler("batch", "batch.profile.manage"))
	for _, path := range []string{"/members", "/members/{membershipID}"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			l, o, e := page(r)
			if e != nil {
				return e
			}
			v, e := m.List(r.Context(), userID(r), batchID(r), param(r, "membershipID"), l, o)
			if e != nil {
				return e
			}
			return send(w, 200, v)
		}))
	}
	r.Post("/members", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			UserID string `json:"user_id"`
		}](w, r)
		if e != nil {
			return e
		}
		id, e := m.Create(r.Context(), userID(r), batchID(r), in.UserID, false)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/roles", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := m.Catalog(r.Context(), userID(r), batchID(r))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Post("/members/{membershipID}/roles", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			Role string `json:"role"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = m.Role(r.Context(), userID(r), batchID(r), param(r, "membershipID"), in.Role, false, false); e != nil {
			return e
		}
		v, e := m.List(r.Context(), userID(r), batchID(r), param(r, "membershipID"), 1, 0)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Delete("/members/{membershipID}/roles/{roleCode}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := m.Role(r.Context(), userID(r), batchID(r), param(r, "membershipID"), param(r, "roleCode"), true, false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	for _, x := range []struct{ path, status string }{{"suspend", "SUSPENDED"}, {"reactivate", "ACTIVE"}, {"graduate", "GRADUATED"}, {"leave", "LEFT"}} {
		r.Post("/members/{membershipID}/"+x.path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			if e := m.SetStatus(r.Context(), userID(r), batchID(r), param(r, "membershipID"), x.status); e != nil {
				return e
			}
			return send(w, 204, nil)
		}))
	}
}
