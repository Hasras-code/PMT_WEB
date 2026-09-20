package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/complaint"
	"github.com/Hasras-code/PMT_WEB.git/internal/feedback"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) supportRoutes(r chi.Router) {
	s := complaint.Service{Pool: a.Pool}
	f := feedback.Service{Pool: a.Pool}
	r.Post("/complaints", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[complaint.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	for _, path := range []string{"/complaints", "/complaints/mine"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			l, o, e := page(r)
			if e != nil {
				return e
			}
			v, e := s.List(r.Context(), userID(r), batchID(r), path == "/complaints/mine", l, o)
			if e != nil {
				return e
			}
			return send(w, 200, v)
		}))
	}
	r.Get("/complaints/{complaintID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "complaintID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Get("/complaints/{complaintID}/messages", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.Messages(r.Context(), userID(r), batchID(r), param(r, "complaintID"), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Post("/complaints/{complaintID}/messages", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			Message string `json:"message"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = s.Reply(r.Context(), userID(r), batchID(r), param(r, "complaintID"), in.Message); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Patch("/complaints/{complaintID}/status", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			Status string `json:"status"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = s.Transition(r.Context(), userID(r), batchID(r), param(r, "complaintID"), in.Status); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Patch("/complaints/{complaintID}/assignee", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			UserID string `json:"user_id"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = s.Assign(r.Context(), userID(r), batchID(r), param(r, "complaintID"), in.UserID); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/feedback", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[feedback.Input](w, r)
		if e != nil {
			return e
		}
		id, e := f.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/feedback", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := f.List(r.Context(), userID(r), batchID(r), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/feedback/{feedbackID}/status", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			Status string `json:"status"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = f.Status(r.Context(), userID(r), batchID(r), param(r, "feedbackID"), in.Status); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
