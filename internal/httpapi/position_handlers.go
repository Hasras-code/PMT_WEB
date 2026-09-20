package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/position"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

func (a *API) positionRoutes(r chi.Router) {
	s := position.Service{Pool: a.Pool}
	for _, path := range []string{"/positions", "/positions/{positionID}"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			l, o, e := page(r)
			if e != nil {
				return e
			}
			v, e := s.List(r.Context(), userID(r), batchID(r), param(r, "positionID"), l, o)
			if e != nil {
				return e
			}
			return send(w, 200, v)
		}))
	}
	for _, route := range []struct{ method, path string }{{"POST", "/positions"}, {"PATCH", "/positions/{positionID}"}} {
		r.MethodFunc(route.method, route.path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			in, e := decode[position.Input](w, r)
			if e != nil {
				return e
			}
			id, e := s.Save(r.Context(), userID(r), batchID(r), param(r, "positionID"), in)
			if e != nil {
				return e
			}
			if r.Method == "POST" {
				return created(w, id)
			}
			return send(w, 204, nil)
		}))
	}
	r.Delete("/positions/{positionID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Hide(r.Context(), userID(r), batchID(r), param(r, "positionID")); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/positions/{positionID}/assignments", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.Assignments(r.Context(), userID(r), batchID(r), param(r, "positionID"), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Post("/positions/{positionID}/assignments", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			UserID string     `json:"user_id"`
			Starts time.Time  `json:"starts_at"`
			Ends   *time.Time `json:"ends_at"`
		}](w, r)
		if e != nil {
			return e
		}
		id, e := s.Assign(r.Context(), userID(r), batchID(r), param(r, "positionID"), in.UserID, in.Starts, in.Ends)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Post("/positions/{positionID}/assignments/{assignmentID}/end", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.End(r.Context(), userID(r), batchID(r), param(r, "positionID"), param(r, "assignmentID")); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
