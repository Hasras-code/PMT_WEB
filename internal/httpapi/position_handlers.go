package httpapi

import (
	"net/http"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/position"
	"github.com/go-chi/chi/v5"
)

func (a *API) positionRoutes(r chi.Router) {
	r.Get("/positions", a.wrap(a.listPositionsHandler))
	r.Get("/positions/{positionID}", a.wrap(a.listPositionHandler))
	r.Post("/positions", a.wrap(a.createPositionHandler))
	r.Patch("/positions/{positionID}", a.wrap(a.updatePositionHandler))
	r.Delete("/positions/{positionID}", a.wrap(a.deletePositionHandler))
	r.Get("/positions/{positionID}/assignments", a.wrap(a.listAssignmentsHandler))
	r.Post("/positions/{positionID}/assignments", a.wrap(a.assignPositionHandler))
	r.Post("/positions/{positionID}/assignments/{assignmentID}/end", a.wrap(a.endAssignmentHandler))
}

// listPositionsHandler godoc
// @Summary List positions
// @Tags positions
// @Produce json
// @Param batchID path string true "Batch ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions [get]
func (a *API) listPositionsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPositions(w, r)
}

// listPositionHandler godoc
// @Summary List a position
// @Tags positions
// @Produce json
// @Param batchID path string true "Batch ID"
// @Param positionID path string true "Position ID"
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions/{positionID} [get]
func (a *API) listPositionHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPositions(w, r)
}

func (a *API) listPositions(w http.ResponseWriter, r *http.Request) error {
	s := position.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.List(r.Context(), userID(r), batchID(r), param(r, "positionID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// createPositionHandler godoc
// @Summary Create a position
// @Tags positions
// @Accept json
// @Param batchID path string true "Batch ID"
// @Param payload body position.Input true "Position"
// @Success 201 {object} map[string]string
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions [post]
func (a *API) createPositionHandler(w http.ResponseWriter, r *http.Request) error {
	return a.savePosition(w, r, true)
}

// updatePositionHandler godoc
// @Summary Update a position
// @Tags positions
// @Accept json
// @Param batchID path string true "Batch ID"
// @Param positionID path string true "Position ID"
// @Param payload body position.Input true "Position"
// @Success 204
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions/{positionID} [patch]
func (a *API) updatePositionHandler(w http.ResponseWriter, r *http.Request) error {
	return a.savePosition(w, r, false)
}

func (a *API) savePosition(w http.ResponseWriter, r *http.Request, create bool) error {
	s := position.Service{Pool: a.Pool}
	in, e := decode[position.Input](w, r)
	if e != nil {
		return e
	}
	id, e := s.Save(r.Context(), userID(r), batchID(r), param(r, "positionID"), in)
	if e != nil {
		return e
	}
	if create {
		return created(w, id)
	}
	return send(w, 204, nil)
}

// deletePositionHandler godoc
// @Summary Hide a position
// @Tags positions
// @Param batchID path string true "Batch ID"
// @Param positionID path string true "Position ID"
// @Success 204
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions/{positionID} [delete]
func (a *API) deletePositionHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (position.Service{Pool: a.Pool}).Hide(r.Context(), userID(r), batchID(r), param(r, "positionID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// listAssignmentsHandler godoc
// @Summary List position assignments
// @Tags positions
// @Produce json
// @Param batchID path string true "Batch ID"
// @Param positionID path string true "Position ID"
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions/{positionID}/assignments [get]
func (a *API) listAssignmentsHandler(w http.ResponseWriter, r *http.Request) error {
	s := position.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.Assignments(r.Context(), userID(r), batchID(r), param(r, "positionID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// assignPositionHandler godoc
// @Summary Assign a user to a position
// @Tags positions
// @Accept json
// @Param batchID path string true "Batch ID"
// @Param positionID path string true "Position ID"
// @Param payload body map[string]any true "Assignment"
// @Success 201 {object} map[string]string
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions/{positionID}/assignments [post]
func (a *API) assignPositionHandler(w http.ResponseWriter, r *http.Request) error {
	s := position.Service{Pool: a.Pool}
	in, e := decode[struct {
		UserID string     `json:"user_id"`
		Starts time.Time  `json:"starts_at"`
		Ends   *time.Time `json:"ends_at"`
	}](w, r)
	if e != nil {
		return e
	}
	id, e := s.Assign(r.Context(), userID(r), batchID(r), param(r, "positionID"), in.UserID, in.Starts, in.Ends)
	if e != nil {
		return e
	}
	return created(w, id)
}

// endAssignmentHandler godoc
// @Summary End a position assignment
// @Tags positions
// @Param batchID path string true "Batch ID"
// @Param positionID path string true "Position ID"
// @Param assignmentID path string true "Assignment ID"
// @Success 204
// @Security bearerAuth
// @Router /v1/batches/{batchID}/positions/{positionID}/assignments/{assignmentID}/end [post]
func (a *API) endAssignmentHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (position.Service{Pool: a.Pool}).End(r.Context(), userID(r), batchID(r), param(r, "positionID"), param(r, "assignmentID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}
