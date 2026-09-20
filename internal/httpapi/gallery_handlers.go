package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/gallery"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (a *API) galleryService() gallery.Service {
	return gallery.Service{Pool: a.Pool, Store: a.Files, BaseURL: a.Config.BaseURL, Secret: []byte(a.Config.Secret)}
}
func (a *API) galleryPublicRoutes(r chi.Router) {
	s := a.galleryService()
	r.Get("/public/gallery", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		limit := 12
		if v := r.URL.Query().Get("limit"); v != "" {
			var e error
			limit, e = strconv.Atoi(v)
			if e != nil {
				return apperror.ErrInvalid
			}
		}
		v, e := s.List(r.Context(), r.URL.Query().Get("month"), r.URL.Query().Get("cursor"), limit)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Get("/public/gallery/{imageID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), param(r, "imageID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Get("/public/gallery/{imageID}/files/{variant}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		o, e := s.File(r.Context(), param(r, "imageID"), param(r, "variant"))
		if e != nil {
			return e
		}
		return a.serveFile(w, r, o.Key, o.MIME, "gallery", true)
	}))
}
func (a *API) galleryAdminRoutes(r chi.Router) {
	s := a.galleryService()
	r.Post("/admin/gallery/uploads", a.uploadHandler("gallery", "gallery.manage"))
	for _, path := range []string{"/admin/gallery", "/admin/gallery/{imageID}"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			l, o, e := page(r)
			if e != nil {
				return e
			}
			v, e := s.AdminList(r.Context(), userID(r), param(r, "imageID"), l, o)
			if e != nil {
				return e
			}
			return send(w, 200, v)
		}))
	}
	r.Post("/admin/gallery", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[gallery.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Patch("/admin/gallery/{imageID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[gallery.Update](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), param(r, "imageID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/admin/gallery/{imageID}/publish", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), param(r, "imageID"), true); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/admin/gallery/{imageID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), param(r, "imageID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
}
