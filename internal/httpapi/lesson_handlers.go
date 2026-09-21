package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/lesson"
	"github.com/go-chi/chi/v5"
)

func (a *API) lessonRoutes(r chi.Router) {
	r.Get("/lessons", a.wrap(a.listLessonsHandler))
	r.Post("/lessons", a.wrap(a.createLessonHandler))
	r.Get("/lessons/{lessonID}", a.wrap(a.getLessonHandler))
	r.Patch("/lessons/{lessonID}", a.wrap(a.updateLessonHandler))
	r.Delete("/lessons/{lessonID}", a.wrap(a.deleteLessonHandler))
	r.Post("/lessons/{lessonID}/publish", a.wrap(a.publishLessonHandler))
}

// listLessonsHandler godoc
//
//	@Summary List lessons
//	@Tags lessons
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/lessons [get]
func (a *API) listLessonsHandler(w http.ResponseWriter, r *http.Request) error {
	s := lesson.Service{Pool: a.Pool}
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

// createLessonHandler godoc
//
//	@Summary Create a lesson
//	@Tags lessons
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body lesson.Input true "Lesson"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/lessons [post]
func (a *API) createLessonHandler(w http.ResponseWriter, r *http.Request) error {
	s := lesson.Service{Pool: a.Pool}
	in, e := decode[lesson.Input](w, r)
	if e != nil {
		return e
	}
	id, e := s.Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// getLessonHandler godoc
//
//	@Summary Get a lesson
//	@Tags lessons
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param lessonID path string true "Lesson ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/lessons/{lessonID} [get]
func (a *API) getLessonHandler(w http.ResponseWriter, r *http.Request) error {
	s := lesson.Service{Pool: a.Pool}
	v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "lessonID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateLessonHandler godoc
//
//	@Summary Update a lesson
//	@Tags lessons
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param lessonID path string true "Lesson ID"
//	@Param payload body lesson.Input true "Lesson"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/lessons/{lessonID} [patch]
func (a *API) updateLessonHandler(w http.ResponseWriter, r *http.Request) error {
	s := lesson.Service{Pool: a.Pool}
	in, e := decode[lesson.Input](w, r)
	if e != nil {
		return e
	}
	if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "lessonID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// deleteLessonHandler godoc
//
//	@Summary Archive a lesson
//	@Tags lessons
//	@Param batchID path string true "Batch ID"
//	@Param lessonID path string true "Lesson ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/lessons/{lessonID} [delete]
func (a *API) deleteLessonHandler(w http.ResponseWriter, r *http.Request) error {
	s := lesson.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "lessonID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// publishLessonHandler godoc
//
//	@Summary Publish a lesson
//	@Tags lessons
//	@Param batchID path string true "Batch ID"
//	@Param lessonID path string true "Lesson ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/lessons/{lessonID}/publish [post]
func (a *API) publishLessonHandler(w http.ResponseWriter, r *http.Request) error {
	s := lesson.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "lessonID"), true); e != nil {
		return e
	}
	return send(w, 204, nil)
}
