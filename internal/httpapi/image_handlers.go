package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/event"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
)

func (a *API) eventImageRoutes(r chi.Router) {
	r.Post("/events/{eventID}/uploads", a.wrap(a.authorizeEventUploadHandler))
	r.Post("/events/{eventID}/image", a.wrap(a.setEventImageHandler))
	r.Get("/events/{eventID}/image", a.wrap(a.getEventImageHandler))
}

// setEventImageHandler godoc
//
//	@Summary Set an event image
//	@Tags images
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Param payload body map[string]string true "Upload ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID}/image [post]
func (a *API) setEventImageHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
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
}

// getEventImageHandler godoc
//
//	@Summary Get an event image
//	@Tags images
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID}/image [get]
func (a *API) getEventImageHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
	o, e := s.Image(r.Context(), userID(r), batchID(r), param(r, "eventID"))
	if e != nil {
		return e
	}
	url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
	if e != nil {
		return e
	}
	return send(w, 200, map[string]any{"url": url, "expires_in": 300})
}
func (a *API) profileImageRoutes(r chi.Router) {
	r.Get("/me/profile/image", a.wrap(a.getProfileImageHandler))
}

// getProfileImageHandler godoc
//
//	@Summary Get the profile image
//	@Tags images
//	@Produce json
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/me/profile/image [get]
func (a *API) getProfileImageHandler(w http.ResponseWriter, r *http.Request) error {
	s := user.Service{Pool: a.Pool}
	o, e := s.Image(r.Context(), userID(r))
	if e != nil {
		return e
	}
	url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
	if e != nil {
		return e
	}
	return send(w, 200, map[string]any{"url": url, "expires_in": 300})
}
func (a *API) publicImageRoutes(r chi.Router) {
	r.Get("/public/batches/{slug}/image", a.wrap(a.getPublicBatchImageHandler))
	r.Get("/public/batches/{slug}/events/{eventID}/image", a.wrap(a.getPublicEventImageHandler))
}

// getPublicBatchImageHandler godoc
//
//	@Summary Get a public batch image
//	@Tags images
//	@Param slug path string true "Batch slug"
//	@Success 200 {file} binary
//	@Router /v1/public/batches/{slug}/image [get]
func (a *API) getPublicBatchImageHandler(w http.ResponseWriter, r *http.Request) error {
	s := batch.Service{Pool: a.Pool}
	return a.servePublicImage(w, r, s)
}

// getPublicEventImageHandler godoc
//
//	@Summary Get a public event image
//	@Tags images
//	@Param slug path string true "Batch slug"
//	@Param eventID path string true "Event ID"
//	@Success 200 {file} binary
//	@Router /v1/public/batches/{slug}/events/{eventID}/image [get]
func (a *API) getPublicEventImageHandler(w http.ResponseWriter, r *http.Request) error {
	s := batch.Service{Pool: a.Pool}
	return a.servePublicImage(w, r, s)
}

func (a *API) servePublicImage(w http.ResponseWriter, r *http.Request, s batch.Service) error {
	o, e := s.PublicImage(r.Context(), param(r, "slug"), param(r, "eventID"))
	if e != nil {
		return e
	}
	return a.serveFile(w, r, o.Key, o.MIME, o.Name, true)
}
