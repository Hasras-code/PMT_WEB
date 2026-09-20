package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/event"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) eventRoutes(r chi.Router) {
	s := event.Service{Pool: a.Pool}
	r.Get("/events", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
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
	r.Post("/events", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[event.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/events/{eventID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "eventID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/events/{eventID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[event.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "eventID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/events/{eventID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "eventID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/events/{eventID}/publish", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "eventID"), true); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
