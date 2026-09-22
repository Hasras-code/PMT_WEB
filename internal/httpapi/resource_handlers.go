package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/announcement"
	"github.com/Hasras-code/PMT_WEB.git/internal/resource"
	"github.com/go-chi/chi/v5"
)

func (a *API) resourceRoutes(r chi.Router) {
	r.Get("/resources", a.wrap(a.listResourcesHandler))
	r.Post("/resources/uploads", a.wrap(a.authorizeResourceUploadHandler))
	r.Post("/resources", a.wrap(a.createResourceHandler))
	r.Get("/resources/{resourceID}", a.wrap(a.getResourceHandler))
	r.Patch("/resources/{resourceID}", a.wrap(a.updateResourceHandler))
	r.Delete("/resources/{resourceID}", a.wrap(a.deleteResourceHandler))
	r.Post("/resources/{resourceID}/publish", a.wrap(a.publishResourceHandler))
	r.Post("/resources/{resourceID}/versions/uploads", a.wrap(a.authorizeResourceVersionUploadHandler))
	r.Post("/resources/{resourceID}/versions", a.wrap(a.addResourceVersionHandler))
	r.Get("/resources/{resourceID}/versions", a.wrap(a.listResourceVersionsHandler))
	r.Get("/resources/{resourceID}/download", a.wrap(a.downloadResourceHandler))
	r.Get("/resources/{resourceID}/versions/{versionID}/download", a.wrap(a.downloadResourceVersionHandler))
	r.Put("/resources/{resourceID}/bookmark", a.wrap(a.bookmarkResourceHandler))
	r.Delete("/resources/{resourceID}/bookmark", a.wrap(a.unbookmarkResourceHandler))
	r.Get("/announcements/{announcementID}/attachments", a.wrap(a.listAttachmentsHandler))

	r.Post("/announcements/{announcementID}/attachments/uploads", a.wrap(a.authorizeAttachmentUploadHandler))
	r.Post("/announcements/{announcementID}/attachments", a.wrap(a.attachAnnouncementHandler))
	r.Delete("/announcements/{announcementID}/attachments/{attachmentID}", a.wrap(a.deleteAttachmentHandler))
	r.Get("/announcements/{announcementID}/attachments/{attachmentID}/download", a.wrap(a.downloadAttachmentHandler))
}

// listResourcesHandler godoc
//
//	@Summary List resources
//	@Tags resources
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param module_id query string false "Filter by subject/module UUID"
//	@Param type query string false "Filter by resource type"
//	@Param query query string false "Search title, description, or subject"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources [get]
func (a *API) listResourcesHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (resource.Service{Pool: a.Pool}).List(r.Context(), userID(r), batchID(r), resource.Filter{
		ModuleID: r.URL.Query().Get("module_id"),
		Type:     r.URL.Query().Get("type"),
		Query:    r.URL.Query().Get("query"),
	}, l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// createResourceHandler godoc
//
//	@Summary Create a resource
//	@Tags resources
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body resource.Input true "Resource"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources [post]
func (a *API) createResourceHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[resource.Input](w, r)
	if e != nil {
		return e
	}
	id, e := (resource.Service{Pool: a.Pool}).Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// getResourceHandler godoc
//
//	@Summary Get a resource
//	@Tags resources
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID} [get]
func (a *API) getResourceHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (resource.Service{Pool: a.Pool}).Get(r.Context(), userID(r), batchID(r), param(r, "resourceID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateResourceHandler godoc
//
//	@Summary Update a resource
//	@Tags resources
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Param payload body resource.Update true "Resource"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID} [patch]
func (a *API) updateResourceHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[resource.Update](w, r)
	if e != nil {
		return e
	}
	if e = (resource.Service{Pool: a.Pool}).Update(r.Context(), userID(r), batchID(r), param(r, "resourceID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// deleteResourceHandler godoc
//
//	@Summary Archive a resource
//	@Tags resources
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID} [delete]
func (a *API) deleteResourceHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (resource.Service{Pool: a.Pool}).Transition(r.Context(), userID(r), batchID(r), param(r, "resourceID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// publishResourceHandler godoc
//
//	@Summary Publish a resource
//	@Tags resources
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/publish [post]
func (a *API) publishResourceHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (resource.Service{Pool: a.Pool}).Transition(r.Context(), userID(r), batchID(r), param(r, "resourceID"), true); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// addResourceVersionHandler godoc
//
//	@Summary Add a resource version
//	@Tags resources
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Param payload body resource.Input true "Resource version"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/versions [post]
func (a *API) addResourceVersionHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[resource.Input](w, r)
	if e != nil {
		return e
	}
	if e = (resource.Service{Pool: a.Pool}).AddVersion(r.Context(), userID(r), batchID(r), param(r, "resourceID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// listResourceVersionsHandler godoc
//
//	@Summary List resource versions
//	@Tags resources
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/versions [get]
func (a *API) listResourceVersionsHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (resource.Service{Pool: a.Pool}).Versions(r.Context(), userID(r), batchID(r), param(r, "resourceID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

func (a *API) downloadResource(w http.ResponseWriter, r *http.Request) error {
	o, e := (resource.Service{Pool: a.Pool}).Download(r.Context(), userID(r), batchID(r), param(r, "resourceID"), param(r, "versionID"))
	if e != nil {
		return e
	}
	url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
	if e != nil {
		return e
	}
	return send(w, 200, map[string]any{"url": url, "expires_in": 300})
}

// downloadResourceHandler godoc
//
//	@Summary Get a resource download URL
//	@Tags resources
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/download [get]
func (a *API) downloadResourceHandler(w http.ResponseWriter, r *http.Request) error {
	return a.downloadResource(w, r)
}

// downloadResourceVersionHandler godoc
//
//	@Summary Get a resource version download URL
//	@Tags resources
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Param versionID path string true "Version ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/versions/{versionID}/download [get]
func (a *API) downloadResourceVersionHandler(w http.ResponseWriter, r *http.Request) error {
	return a.downloadResource(w, r)
}

func (a *API) bookmarkResource(w http.ResponseWriter, r *http.Request, remove bool) error {
	if e := (resource.Service{Pool: a.Pool}).Bookmark(r.Context(), userID(r), batchID(r), param(r, "resourceID"), remove); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// bookmarkResourceHandler godoc
//
//	@Summary Bookmark a resource
//	@Tags resources
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/bookmark [put]
func (a *API) bookmarkResourceHandler(w http.ResponseWriter, r *http.Request) error {
	return a.bookmarkResource(w, r, false)
}

// unbookmarkResourceHandler godoc
//
//	@Summary Remove a resource bookmark
//	@Tags resources
//	@Param batchID path string true "Batch ID"
//	@Param resourceID path string true "Resource ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/resources/{resourceID}/bookmark [delete]
func (a *API) unbookmarkResourceHandler(w http.ResponseWriter, r *http.Request) error {
	return a.bookmarkResource(w, r, true)
}

// listAttachmentsHandler godoc
//
//	@Summary List announcement attachments
//	@Tags announcements
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param announcementID path string true "Announcement ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/announcements/{announcementID}/attachments [get]
func (a *API) listAttachmentsHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (announcement.Service{Pool: a.Pool}).Attachments(r.Context(), userID(r), batchID(r), param(r, "announcementID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// attachAnnouncementHandler godoc
//
//	@Summary Attach a file to an announcement
//	@Tags announcements
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param announcementID path string true "Announcement ID"
//	@Param payload body map[string]string true "Upload ID"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/announcements/{announcementID}/attachments [post]
func (a *API) attachAnnouncementHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		UploadID string `json:"upload_id"`
	}](w, r)
	if e != nil {
		return e
	}
	id, e := (announcement.Service{Pool: a.Pool}).Attach(r.Context(), userID(r), batchID(r), param(r, "announcementID"), in.UploadID)
	if e != nil {
		return e
	}
	return created(w, id)
}

// deleteAttachmentHandler godoc
//
//	@Summary Remove an announcement attachment
//	@Tags announcements
//	@Param batchID path string true "Batch ID"
//	@Param announcementID path string true "Announcement ID"
//	@Param attachmentID path string true "Attachment ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/announcements/{announcementID}/attachments/{attachmentID} [delete]
func (a *API) deleteAttachmentHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (announcement.Service{Pool: a.Pool}).RemoveAttachment(r.Context(), userID(r), batchID(r), param(r, "announcementID"), param(r, "attachmentID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// downloadAttachmentHandler godoc
//
//	@Summary Get an announcement attachment download URL
//	@Tags announcements
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param announcementID path string true "Announcement ID"
//	@Param attachmentID path string true "Attachment ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/announcements/{announcementID}/attachments/{attachmentID}/download [get]
func (a *API) downloadAttachmentHandler(w http.ResponseWriter, r *http.Request) error {
	o, e := (announcement.Service{Pool: a.Pool}).Attachment(r.Context(), userID(r), batchID(r), param(r, "announcementID"), param(r, "attachmentID"))
	if e != nil {
		return e
	}
	url, e := a.Files.DownloadURL(a.Config.BaseURL, o.Key, o.Name, o.MIME)
	if e != nil {
		return e
	}
	return send(w, 200, map[string]any{"url": url, "expires_in": 300})
}
