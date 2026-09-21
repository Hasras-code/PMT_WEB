package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/module"
	"github.com/go-chi/chi/v5"
)

func (a *API) moduleRoutes(r chi.Router) {
	r.Get("/modules", a.wrap(a.listModulesHandler))
	r.Post("/modules", a.wrap(a.createModuleHandler))
	r.Get("/modules/{moduleID}", a.wrap(a.getModuleHandler))
	r.Patch("/modules/{moduleID}", a.wrap(a.updateModuleHandler))
	r.Delete("/modules/{moduleID}", a.wrap(a.deleteModuleHandler))
}

// listModulesHandler godoc
//
//	@Summary List modules
//	@Tags modules
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/modules [get]
func (a *API) listModulesHandler(w http.ResponseWriter, r *http.Request) error {
	s := module.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.List(r.Context(), userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// createModuleHandler godoc
//
//	@Summary Create a module
//	@Tags modules
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body module.Input true "Module"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/modules [post]
func (a *API) createModuleHandler(w http.ResponseWriter, r *http.Request) error {
	s := module.Service{Pool: a.Pool}
	in, e := decode[module.Input](w, r)
	if e != nil {
		return e
	}
	id, e := s.Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// getModuleHandler godoc
//
//	@Summary Get a module
//	@Tags modules
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param moduleID path string true "Module ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/modules/{moduleID} [get]
func (a *API) getModuleHandler(w http.ResponseWriter, r *http.Request) error {
	s := module.Service{Pool: a.Pool}
	v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "moduleID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateModuleHandler godoc
//
//	@Summary Update a module
//	@Tags modules
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param moduleID path string true "Module ID"
//	@Param payload body module.Input true "Module"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/modules/{moduleID} [patch]
func (a *API) updateModuleHandler(w http.ResponseWriter, r *http.Request) error {
	s := module.Service{Pool: a.Pool}
	in, e := decode[module.Input](w, r)
	if e != nil {
		return e
	}
	if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "moduleID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// deleteModuleHandler godoc
//
//	@Summary Archive a module
//	@Tags modules
//	@Param batchID path string true "Batch ID"
//	@Param moduleID path string true "Module ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/modules/{moduleID} [delete]
func (a *API) deleteModuleHandler(w http.ResponseWriter, r *http.Request) error {
	s := module.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "moduleID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
}
