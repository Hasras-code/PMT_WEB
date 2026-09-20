package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/announcement"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) announcementRoutes(r chi.Router) {
	s := announcement.Service{Pool: a.Pool}
	r.Get("/announcements", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
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
	r.Post("/announcements", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[announcement.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/announcements/{announcementID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "announcementID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/announcements/{announcementID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[announcement.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "announcementID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/announcements/{announcementID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "announcementID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/announcements/{announcementID}/publish", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "announcementID"), true); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
