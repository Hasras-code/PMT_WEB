package httpapi

import (
	"net/http"
	"strconv"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/gallery"
	"github.com/go-chi/chi/v5"
)

func (a *API) galleryService() gallery.Service {
	return gallery.Service{Pool: a.Pool, Store: a.Files, BaseURL: a.Config.BaseURL, Secret: []byte(a.Config.Secret)}
}
func (a *API) galleryPublicRoutes(r chi.Router) {
	r.Get("/public/gallery", a.wrap(a.listPublicGalleryHandler))
	r.Get("/public/gallery/{imageID}", a.wrap(a.getPublicGalleryImageHandler))
	r.Get("/public/gallery/{imageID}/files/{variant}", a.wrap(a.getPublicGalleryFileHandler))
}
func (a *API) galleryAdminRoutes(r chi.Router) {
	r.Post("/admin/gallery/uploads", a.wrap(a.authorizeGalleryUploadHandler))
	r.Get("/admin/gallery", a.wrap(a.listAdminGalleryHandler))
	r.Get("/admin/gallery/{imageID}", a.wrap(a.getAdminGalleryHandler))
	r.Post("/admin/gallery", a.wrap(a.createGalleryHandler))
	r.Patch("/admin/gallery/{imageID}", a.wrap(a.updateGalleryHandler))
	r.Post("/admin/gallery/{imageID}/publish", a.wrap(a.publishGalleryHandler))
	r.Delete("/admin/gallery/{imageID}", a.wrap(a.unpublishGalleryHandler))
}

// listPublicGalleryHandler godoc
//
//	@Summary List public gallery images
//	@Tags gallery
//	@Produce json
//	@Param month query string false "Month"
//	@Param cursor query string false "Cursor"
//	@Param limit query int false "Limit"
//	@Success 200 {array} map[string]any
//	@Router /v1/public/gallery [get]
func (a *API) listPublicGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	limit := 12
	if value := r.URL.Query().Get("limit"); value != "" {
		var e error
		limit, e = strconv.Atoi(value)
		if e != nil {
			return apperror.ErrInvalid
		}
	}
	v, e := a.galleryService().List(r.Context(), r.URL.Query().Get("month"), r.URL.Query().Get("cursor"), limit)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// getPublicGalleryImageHandler godoc
//
//	@Summary Get a public gallery image
//	@Tags gallery
//	@Produce json
//	@Param imageID path string true "Image ID"
//	@Success 200 {object} map[string]any
//	@Router /v1/public/gallery/{imageID} [get]
func (a *API) getPublicGalleryImageHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := a.galleryService().Get(r.Context(), param(r, "imageID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// getPublicGalleryFileHandler godoc
//
//	@Summary Download a public gallery file
//	@Tags gallery
//	@Param imageID path string true "Image ID"
//	@Param variant path string true "File variant"
//	@Success 200 {file} binary
//	@Router /v1/public/gallery/{imageID}/files/{variant} [get]
func (a *API) getPublicGalleryFileHandler(w http.ResponseWriter, r *http.Request) error {
	o, e := a.galleryService().File(r.Context(), param(r, "imageID"), param(r, "variant"))
	if e != nil {
		return e
	}
	return a.serveFile(w, r, o.Key, o.MIME, "gallery", true)
}

func (a *API) listAdminGallery(w http.ResponseWriter, r *http.Request, imageID string) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := a.galleryService().AdminList(r.Context(), userID(r), imageID, l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// listAdminGalleryHandler godoc
//
//	@Summary List gallery images
//	@Tags gallery
//	@Produce json
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/admin/gallery [get]
func (a *API) listAdminGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listAdminGallery(w, r, "")
}

// getAdminGalleryHandler godoc
//
//	@Summary Get a gallery image in the admin view
//	@Tags gallery
//	@Produce json
//	@Param imageID path string true "Image ID"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/admin/gallery/{imageID} [get]
func (a *API) getAdminGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listAdminGallery(w, r, param(r, "imageID"))
}

// createGalleryHandler godoc
//
//	@Summary Create a gallery image
//	@Tags gallery
//	@Accept json
//	@Param payload body gallery.Input true "Gallery image"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/admin/gallery [post]
func (a *API) createGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[gallery.Input](w, r)
	if e != nil {
		return e
	}
	id, e := a.galleryService().Create(r.Context(), userID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// updateGalleryHandler godoc
//
//	@Summary Update a gallery image
//	@Tags gallery
//	@Accept json
//	@Param imageID path string true "Image ID"
//	@Param payload body gallery.Update true "Gallery image"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/gallery/{imageID} [patch]
func (a *API) updateGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[gallery.Update](w, r)
	if e != nil {
		return e
	}
	if e = a.galleryService().Update(r.Context(), userID(r), param(r, "imageID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// publishGalleryHandler godoc
//
//	@Summary Publish a gallery image
//	@Tags gallery
//	@Param imageID path string true "Image ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/gallery/{imageID}/publish [post]
func (a *API) publishGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	if e := a.galleryService().Transition(r.Context(), userID(r), param(r, "imageID"), true); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// unpublishGalleryHandler godoc
//
//	@Summary Remove a gallery image
//	@Tags gallery
//	@Param imageID path string true "Image ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/gallery/{imageID} [delete]
func (a *API) unpublishGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	if e := a.galleryService().Transition(r.Context(), userID(r), param(r, "imageID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
}
