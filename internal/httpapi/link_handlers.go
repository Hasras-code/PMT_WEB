package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/link"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) linkRoutes(r chi.Router) {
	s := link.Service{Pool: a.Pool}
	r.Get("/links", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
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
	r.Post("/links", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[link.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/links/{linkID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "linkID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/links/{linkID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[link.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "linkID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/links/{linkID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "linkID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/links/{linkID}/publish", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "linkID"), true); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
