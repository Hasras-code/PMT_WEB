package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/lesson"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) lessonRoutes(r chi.Router) {
	s := lesson.Service{Pool: a.Pool}
	r.Get("/lessons", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
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
	r.Post("/lessons", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[lesson.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/lessons/{lessonID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "lessonID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/lessons/{lessonID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[lesson.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "lessonID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/lessons/{lessonID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "lessonID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/lessons/{lessonID}/publish", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "lessonID"), true); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
