package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/kuppi"
	"github.com/go-chi/chi/v5"
)

func (a *API) kuppiRoutes(r chi.Router) {
	r.Get("/kuppis", a.wrap(a.listKuppisHandler))
	r.Post("/kuppis", a.wrap(a.createKuppiHandler))
	r.Get("/kuppis/{kuppiID}", a.wrap(a.getKuppiHandler))
	r.Patch("/kuppis/{kuppiID}", a.wrap(a.updateKuppiHandler))
	r.Post("/kuppis/{kuppiID}/publish", a.wrap(a.publishKuppiHandler))
	r.Post("/kuppis/{kuppiID}/archive", a.wrap(a.archiveKuppiHandler))
}

// listKuppisHandler godoc
//
//	@Summary List Kuppi recordings
//	@Tags kuppis
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param module_id query string false "Module ID"
//	@Param search query string false "Search title, description, or module"
//	@Param date_from query string false "Recorded-at lower bound"
//	@Param date_to query string false "Recorded-at upper bound"
//	@Param include_archived query bool false "Include archived recordings"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/kuppis [get]
func (a *API) listKuppisHandler(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := page(r)
	if err != nil {
		return err
	}
	value, err := (kuppi.Service{Pool: a.Pool}).List(r.Context(), userID(r), batchID(r), kuppi.Filter{
		ModuleID: r.URL.Query().Get("module_id"), Search: r.URL.Query().Get("search"),
		DateFrom: r.URL.Query().Get("date_from"), DateTo: r.URL.Query().Get("date_to"),
		IncludeArchived: r.URL.Query().Get("include_archived") == "true",
	}, limit, offset)
	if err != nil {
		return err
	}
	return send(w, 200, value)
}

// createKuppiHandler godoc
//
//	@Summary Create a Kuppi recording
//	@Tags kuppis
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body kuppi.Input true "Kuppi recording"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/kuppis [post]
func (a *API) createKuppiHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[kuppi.Input](w, r)
	if err != nil {
		return err
	}
	id, err := (kuppi.Service{Pool: a.Pool}).Create(r.Context(), userID(r), batchID(r), in)
	if err != nil {
		return err
	}
	return created(w, id)
}

// getKuppiHandler godoc
//
//	@Summary Get a Kuppi recording
//	@Tags kuppis
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param kuppiID path string true "Kuppi ID"
//	@Param include_archived query bool false "Include archived recording"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/kuppis/{kuppiID} [get]
func (a *API) getKuppiHandler(w http.ResponseWriter, r *http.Request) error {
	value, err := (kuppi.Service{Pool: a.Pool}).Get(r.Context(), userID(r), batchID(r), param(r, "kuppiID"), r.URL.Query().Get("include_archived") == "true")
	if err != nil {
		return err
	}
	return send(w, 200, value)
}

// updateKuppiHandler godoc
//
//	@Summary Update a Kuppi recording
//	@Tags kuppis
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param kuppiID path string true "Kuppi ID"
//	@Param payload body kuppi.Input true "Kuppi recording"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/kuppis/{kuppiID} [patch]
func (a *API) updateKuppiHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[kuppi.Input](w, r)
	if err != nil {
		return err
	}
	if err = (kuppi.Service{Pool: a.Pool}).Update(r.Context(), userID(r), batchID(r), param(r, "kuppiID"), in); err != nil {
		return err
	}
	return send(w, 204, nil)
}

func (a *API) publishKuppiHandler(w http.ResponseWriter, r *http.Request) error {
	if err := (kuppi.Service{Pool: a.Pool}).Publish(r.Context(), userID(r), batchID(r), param(r, "kuppiID")); err != nil {
		return err
	}
	return send(w, 204, nil)
}

func (a *API) archiveKuppiHandler(w http.ResponseWriter, r *http.Request) error {
	if err := (kuppi.Service{Pool: a.Pool}).Archive(r.Context(), userID(r), batchID(r), param(r, "kuppiID")); err != nil {
		return err
	}
	return send(w, 204, nil)
}
