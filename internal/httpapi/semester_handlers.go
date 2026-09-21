package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/semester"
	"github.com/go-chi/chi/v5"
)

func (a *API) semesterRoutes(r chi.Router) {
	r.Get("/semesters", a.wrap(a.listSemestersHandler))
	r.Post("/semesters", a.wrap(a.createSemesterHandler))
	r.Get("/semesters/{semesterID}", a.wrap(a.getSemesterHandler))
	r.Patch("/semesters/{semesterID}", a.wrap(a.updateSemesterHandler))
	r.Post("/semesters/{semesterID}/set-current", a.wrap(a.setCurrentSemesterHandler))
}

// listSemestersHandler godoc
//
//	@Summary List semesters
//	@Tags semesters
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/semesters [get]
func (a *API) listSemestersHandler(w http.ResponseWriter, r *http.Request) error {
	s := semester.Service{Pool: a.Pool}
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

// createSemesterHandler godoc
//
//	@Summary Create a semester
//	@Tags semesters
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body semester.Input true "Semester"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/semesters [post]
func (a *API) createSemesterHandler(w http.ResponseWriter, r *http.Request) error {
	s := semester.Service{Pool: a.Pool}
	in, e := decode[semester.Input](w, r)
	if e != nil {
		return e
	}
	id, e := s.Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// getSemesterHandler godoc
//
//	@Summary Get a semester
//	@Tags semesters
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param semesterID path string true "Semester ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/semesters/{semesterID} [get]
func (a *API) getSemesterHandler(w http.ResponseWriter, r *http.Request) error {
	s := semester.Service{Pool: a.Pool}
	v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "semesterID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateSemesterHandler godoc
//
//	@Summary Update a semester
//	@Tags semesters
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param semesterID path string true "Semester ID"
//	@Param payload body semester.Input true "Semester"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/semesters/{semesterID} [patch]
func (a *API) updateSemesterHandler(w http.ResponseWriter, r *http.Request) error {
	s := semester.Service{Pool: a.Pool}
	in, e := decode[semester.Input](w, r)
	if e != nil {
		return e
	}
	if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "semesterID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// setCurrentSemesterHandler godoc
//
//	@Summary Set the current semester
//	@Tags semesters
//	@Param batchID path string true "Batch ID"
//	@Param semesterID path string true "Semester ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/semesters/{semesterID}/set-current [post]
func (a *API) setCurrentSemesterHandler(w http.ResponseWriter, r *http.Request) error {
	s := semester.Service{Pool: a.Pool}
	if e := s.SetCurrent(r.Context(), userID(r), batchID(r), param(r, "semesterID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}
