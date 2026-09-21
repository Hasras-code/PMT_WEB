package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/link"
	"github.com/go-chi/chi/v5"
)

func (a *API) linkRoutes(r chi.Router) {
	r.Get("/links", a.wrap(a.listLinksHandler))
	r.Post("/links", a.wrap(a.createLinkHandler))
	r.Get("/links/{linkID}", a.wrap(a.getLinkHandler))
	r.Patch("/links/{linkID}", a.wrap(a.updateLinkHandler))
	r.Delete("/links/{linkID}", a.wrap(a.deleteLinkHandler))
	r.Post("/links/{linkID}/publish", a.wrap(a.publishLinkHandler))
}

// listLinksHandler godoc
//
//	@Summary List links
//	@Tags links
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/links [get]
func (a *API) listLinksHandler(w http.ResponseWriter, r *http.Request) error {
	s := link.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.List(r.Context(), userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
	return send(w, 200, v)
}

// createLinkHandler godoc
//
//	@Summary Create a link
//	@Tags links
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body link.Input true "Link"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/links [post]
func (a *API) createLinkHandler(w http.ResponseWriter, r *http.Request) error {
	s := link.Service{Pool: a.Pool}
	in, e := decode[link.Input](w, r)
	if e != nil {
		return e
	}
	id, e := s.Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
	return created(w, id)
}

// getLinkHandler godoc
//
//	@Summary Get a link
//	@Tags links
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param linkID path string true "Link ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/links/{linkID} [get]
func (a *API) getLinkHandler(w http.ResponseWriter, r *http.Request) error {
	s := link.Service{Pool: a.Pool}
	v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "linkID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
	return send(w, 200, v)
}

// updateLinkHandler godoc
//
//	@Summary Update a link
//	@Tags links
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param linkID path string true "Link ID"
//	@Param payload body link.Input true "Link"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/links/{linkID} [patch]
func (a *API) updateLinkHandler(w http.ResponseWriter, r *http.Request) error {
	s := link.Service{Pool: a.Pool}
	in, e := decode[link.Input](w, r)
	if e != nil {
		return e
	}
	if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "linkID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}

// deleteLinkHandler godoc
//
//	@Summary Archive a link
//	@Tags links
//	@Param batchID path string true "Batch ID"
//	@Param linkID path string true "Link ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/links/{linkID} [delete]
func (a *API) deleteLinkHandler(w http.ResponseWriter, r *http.Request) error {
	s := link.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "linkID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}

// publishLinkHandler godoc
//
//	@Summary Publish a link
//	@Tags links
//	@Param batchID path string true "Batch ID"
//	@Param linkID path string true "Link ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/links/{linkID}/publish [post]
func (a *API) publishLinkHandler(w http.ResponseWriter, r *http.Request) error {
	s := link.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "linkID"), true); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}
