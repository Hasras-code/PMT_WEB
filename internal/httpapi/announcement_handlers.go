package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/announcement"
	"github.com/go-chi/chi/v5"
)

func (a *API) announcementRoutes(r chi.Router) {
	r.Get("/announcements", a.wrap(a.listAnnouncementsHandler))
	r.Post("/announcements", a.wrap(a.createAnnouncementHandler))
	r.Get("/announcements/{announcementID}", a.wrap(a.getAnnouncementHandler))
	r.Patch("/announcements/{announcementID}", a.wrap(a.updateAnnouncementHandler))
	r.Delete("/announcements/{announcementID}", a.wrap(a.deleteAnnouncementHandler))
	r.Post("/announcements/{announcementID}/publish", a.wrap(a.publishAnnouncementHandler))
}

// listAnnouncementsHandler godoc
//
//	@Summary		List announcements
//	@Description	Returns announcements for a batch.
//	@Tags			announcements
//	@Produce		json
//	@Param			batchID	path	string	true	"Batch ID"
//	@Param			limit	query	int	false	"Limit"
//	@Param			offset	query	int	false	"Offset"
//	@Success		200		{array}	map[string]any
//	@Security		bearerAuth
//	@Router			/v1/batches/{batchID}/announcements [get]
func (a *API) listAnnouncementsHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (announcement.Service{Pool: a.Pool}).List(r.Context(), userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// createAnnouncementHandler godoc
//
//	@Summary		Create an announcement
//	@Tags			announcements
//	@Accept			json
//	@Param			batchID	path	string	true	"Batch ID"
//	@Param			payload	body	announcement.Input	true	"Announcement"
//	@Success		201		{object}	map[string]string
//	@Security		bearerAuth
//	@Router			/v1/batches/{batchID}/announcements [post]
func (a *API) createAnnouncementHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[announcement.Input](w, r)
	if e != nil {
		return e
	}
	id, e := (announcement.Service{Pool: a.Pool}).Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// getAnnouncementHandler godoc
//
//	@Summary		Get an announcement
//	@Tags			announcements
//	@Produce		json
//	@Param			batchID	path	string	true	"Batch ID"
//	@Param			announcementID	path	string	true	"Announcement ID"
//	@Success		200		{object}	map[string]any
//	@Security		bearerAuth
//	@Router			/v1/batches/{batchID}/announcements/{announcementID} [get]
func (a *API) getAnnouncementHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (announcement.Service{Pool: a.Pool}).Get(r.Context(), userID(r), batchID(r), param(r, "announcementID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateAnnouncementHandler godoc
//
//	@Summary		Update an announcement
//	@Tags			announcements
//	@Accept			json
//	@Param			batchID	path	string	true	"Batch ID"
//	@Param			announcementID	path	string	true	"Announcement ID"
//	@Param			payload	body	announcement.Input	true	"Announcement"
//	@Success		204
//	@Security		bearerAuth
//	@Router			/v1/batches/{batchID}/announcements/{announcementID} [patch]
func (a *API) updateAnnouncementHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[announcement.Input](w, r)
	if e != nil {
		return e
	}
	if e = (announcement.Service{Pool: a.Pool}).Update(r.Context(), userID(r), batchID(r), param(r, "announcementID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// deleteAnnouncementHandler godoc
//
//	@Summary		Archive an announcement
//	@Tags			announcements
//	@Param			batchID	path	string	true	"Batch ID"
//	@Param			announcementID	path	string	true	"Announcement ID"
//	@Success		204
//	@Security		bearerAuth
//	@Router			/v1/batches/{batchID}/announcements/{announcementID} [delete]
func (a *API) deleteAnnouncementHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (announcement.Service{Pool: a.Pool}).Transition(r.Context(), userID(r), batchID(r), param(r, "announcementID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// publishAnnouncementHandler godoc
//
//	@Summary		Publish an announcement
//	@Tags			announcements
//	@Param			batchID	path	string	true	"Batch ID"
//	@Param			announcementID	path	string	true	"Announcement ID"
//	@Success		204
//	@Security		bearerAuth
//	@Router			/v1/batches/{batchID}/announcements/{announcementID}/publish [post]
func (a *API) publishAnnouncementHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (announcement.Service{Pool: a.Pool}).Transition(r.Context(), userID(r), batchID(r), param(r, "announcementID"), true); e != nil {
		return e
	}
	return send(w, 204, nil)
}
