package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/announcement"
	"github.com/Hasras-code/PMT_WEB.git/internal/resource"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (a *API) resourceRoutes(r chi.Router) {
	s := resource.Service{Pool: a.Pool}
	r.Get("/resources", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
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
	r.Post("/resources/uploads", a.uploadHandler("resource", "resource.create"))
	r.Post("/resources", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[resource.Input](w, r)
		if e != nil {
			return e
		}
		id, e := s.Create(r.Context(), userID(r), batchID(r), in)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Get("/resources/{resourceID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "resourceID"))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/resources/{resourceID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[resource.Update](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "resourceID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Delete("/resources/{resourceID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "resourceID"), false); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/resources/{resourceID}/publish", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "resourceID"), true); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Post("/resources/{resourceID}/versions/uploads", a.uploadHandler("resource", "resource.update"))
	r.Post("/resources/{resourceID}/versions", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[resource.Input](w, r)
		if e != nil {
			return e
		}
		if e = s.AddVersion(r.Context(), userID(r), batchID(r), param(r, "resourceID"), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/resources/{resourceID}/versions", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.Versions(r.Context(), userID(r), batchID(r), param(r, "resourceID"), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	for _, path := range []string{"/resources/{resourceID}/download", "/resources/{resourceID}/versions/{versionID}/download"} {
		r.Get(path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			o, e := s.Download(r.Context(), userID(r), batchID(r), param(r, "resourceID"), param(r, "versionID"))
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
	for _, method := range []string{"PUT", "DELETE"} {
		r.MethodFunc(method, "/resources/{resourceID}/bookmark", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			if e := s.Bookmark(r.Context(), userID(r), batchID(r), param(r, "resourceID"), r.Method == "DELETE"); e != nil {
				return e
			}
			return send(w, 204, nil)
		}))
	}
	as := announcement.Service{Pool: a.Pool}
	r.Get("/announcements/{announcementID}/attachments", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := as.Attachments(r.Context(), userID(r), batchID(r), param(r, "announcementID"), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))

	r.Post("/announcements/{announcementID}/attachments/uploads", a.uploadHandler("attachment", "announcement.update"))
	r.Post("/announcements/{announcementID}/attachments", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			UploadID string `json:"upload_id"`
		}](w, r)
		if e != nil {
			return e
		}
		id, e := as.Attach(r.Context(), userID(r), batchID(r), param(r, "announcementID"), in.UploadID)
		if e != nil {
			return e
		}
		return created(w, id)
	}))
	r.Delete("/announcements/{announcementID}/attachments/{attachmentID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := as.RemoveAttachment(r.Context(), userID(r), batchID(r), param(r, "announcementID"), param(r, "attachmentID")); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/announcements/{announcementID}/attachments/{attachmentID}/download", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		o, e := as.Attachment(r.Context(), userID(r), batchID(r), param(r, "announcementID"), param(r, "attachmentID"))
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
