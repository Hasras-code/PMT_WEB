package httpapi

import (
	"mime"
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/go-chi/chi/v5"
)

// authorizeUploadHandler godoc
//
//	@Summary Authorize a file upload
//	@Description Creates an authorized upload session for the requested resource.
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Failure 422 {object} map[string]any
func (a *API) authorizeUploadHandler(w http.ResponseWriter, r *http.Request, purpose, permission string) error {
	in, e := decode[upload.Input](w, r)
	if e != nil {
		return e
	}
	v, e := a.Uploads.Authorize(r.Context(), userID(r), batchID(r), purpose, permission, in)
	if e != nil {
		return e
	}
	return send(w, 201, v)
}

// authorizeProfileUploadHandler godoc
//
//	@Summary Authorize a profile image upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/me/profile/image/uploads [post]
func (a *API) authorizeProfileUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "profile", "")
}

// authorizeBatchProfileUploadHandler godoc
//
//	@Summary Authorize a batch profile upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/profile/uploads [post]
func (a *API) authorizeBatchProfileUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "batch", "batch.profile.manage")
}

// authorizeGalleryUploadHandler godoc
//
//	@Summary Authorize a gallery image upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/admin/gallery/uploads [post]
func (a *API) authorizeGalleryUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "gallery", "gallery.manage")
}

// authorizeResourceUploadHandler godoc
//
//	@Summary Authorize a resource upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/uploads [post]
func (a *API) authorizeResourceUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "resource", "resource.create")
}

// authorizeResourceVersionUploadHandler godoc
//
//	@Summary Authorize a resource version upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/versions/uploads [post]
func (a *API) authorizeResourceVersionUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "resource", "resource.update")
}

// authorizeAttachmentUploadHandler godoc
//
//	@Summary Authorize an announcement attachment upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param announcementID path string true "Announcement ID"
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/announcements/{announcementID}/attachments/uploads [post]
func (a *API) authorizeAttachmentUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "attachment", "announcement.update")
}

// authorizeEventUploadHandler godoc
//
//	@Summary Authorize an event image upload
//	@Tags files
//	@Accept json
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Param payload body upload.Input true "Upload details"
//	@Success 201 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID}/uploads [post]
func (a *API) authorizeEventUploadHandler(w http.ResponseWriter, r *http.Request) error {
	return a.authorizeUploadHandler(w, r, "event", "event.manage")
}
func (a *API) fileRoutes(r chi.Router) {
	r.Put("/files/uploads/{token}", a.wrap(a.receiveFileHandler))
	r.Get("/files/downloads/{token}", a.wrap(a.downloadFileHandler))
}

// receiveFileHandler godoc
//
//	@Summary Receive an uploaded file
//	@Tags files
//	@Param token path string true "Upload token"
//	@Success 204
//	@Failure 400 {object} map[string]any
//	@Router /v1/files/uploads/{token} [put]
func (a *API) receiveFileHandler(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, (50<<20)+1)
	if e := a.Uploads.Receive(r.Context(), param(r, "token"), r.Header.Get("Content-Type"), r.ContentLength, r.Body); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}

// downloadFileHandler godoc
//
//	@Summary Download a file
//	@Tags files
//	@Param token path string true "Download token"
//	@Success 200 {file} binary
//	@Failure 404 {object} map[string]any
//	@Router /v1/files/downloads/{token} [get]
func (a *API) downloadFileHandler(w http.ResponseWriter, r *http.Request) error {
	g, e := a.Files.Verify(param(r, "token"), "download")
	if e != nil {
		return e
	}
	return a.serveFile(w, r, g.Key, g.MIME, g.Name, false)
	return a.serveFile(w, r, g.Key, g.MIME, g.Name, false)
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
