package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/semester"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) semesterRoutes(r chi.Router) {
	s := semester.Service{Pool: a.Pool}
	r.Get("/semesters", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
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
	r.Post("/semesters", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[semester.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/semesters/{semesterID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "semesterID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/semesters/{semesterID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[semester.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "semesterID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/semesters/{semesterID}/set-current", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.SetCurrent(r.Context(), userID(r), batchID(r), param(r, "semesterID")); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
