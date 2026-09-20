package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/event"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) eventImageRoutes(r chi.Router) {
	s := event.Service{Pool: a.Pool}
	r.Post("/events/{eventID}/uploads", a.uploadHandler("event", "event.manage"))
	r.Post("/events/{eventID}/image", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			UploadID string `json:"upload_id"`
		}](w, r)
		if e != nil {
			return e
		}
		if e = s.SetImage(r.Context(), userID(r), batchID(r), param(r, "eventID"), in.UploadID); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/events/{eventID}/image", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		o, e := s.Image(r.Context(), userID(r), batchID(r), param(r, "eventID"))
		if e != nil {
			return e
		}
		url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
		if e != nil {
			return e
		}
		return send(w, 200, map[string]any{"url": url, "expires_in": 300})
	}))
}
func (a *API) profileImageRoutes(r chi.Router) {
	s := user.Service{Pool: a.Pool}
	r.Get("/me/profile/image", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		o, e := s.Image(r.Context(), userID(r))
		if e != nil {
			return e
		}
		url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
		if e != nil {
			return e
		}
		return send(w, 200, map[string]any{"url": url, "expires_in": 300})
	}))
}
func (a *API) publicImageRoutes(r chi.Router) {
	s := batch.Service{Pool: a.Pool}
	for _, path := range []string{"/public/batches/{slug}/image", "/public/batches/{slug}/events/{eventID}/image"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			o, e := s.PublicImage(r.Context(), param(r, "slug"), param(r, "eventID"))
			if e != nil {
				return e
			}
			return a.serveFile(w, r, o.Key, o.MIME, o.Name, true)
		}))
	}
}
