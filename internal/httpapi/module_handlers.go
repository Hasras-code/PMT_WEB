package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/module"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) moduleRoutes(r chi.Router) {
	s := module.Service{Pool: a.Pool}
	r.Get("/modules", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.List(r.Context(), userID(r), batchID(r), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Post("/modules", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[module.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/modules/{moduleID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "moduleID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/modules/{moduleID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[module.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "moduleID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/modules/{moduleID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "moduleID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
