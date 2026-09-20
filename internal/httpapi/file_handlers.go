package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/go-chi/chi/v5"
	"mime"
	"net/http"
)

func (a *API) uploadHandler(purpose, permission string) http.HandlerFunc {
	return a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[upload.Input](w, r)
		if e != nil {
			return e
		}
		v, e := a.Uploads.Authorize(r.Context(), userID(r), batchID(r), purpose, permission, in)
		if e != nil {
			return e
		}
		return send(w, 201, v)
	})
}
func (a *API) fileRoutes(r chi.Router) {
	r.Put("/files/uploads/{token}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		r.Body = http.MaxBytesReader(w, r.Body, (50<<20)+1)
		if e := a.Uploads.Receive(r.Context(), param(r, "token"), r.Header.Get("Content-Type"), r.ContentLength, r.Body); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/files/downloads/{token}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		g, e := a.Files.Verify(param(r, "token"), "download")
		if e != nil {
			return e
		}
		return a.serveFile(w, r, g.Key, g.MIME, g.Name, false)
	}))
}
func (a *API) serveFile(w http.ResponseWriter, r *http.Request, key, contentType, name string, inline bool) error {
	f, e := a.Files.Read(key)
	if e != nil {
		return e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return e
	}
	w.Header().Set("Content-Type", contentType)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	http.ServeContent(w, r, name, st.ModTime(), f)
	return nil
}
