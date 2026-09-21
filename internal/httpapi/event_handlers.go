package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/event"
	"github.com/go-chi/chi/v5"
)

func (a *API) eventRoutes(r chi.Router) {
	r.Get("/events", a.wrap(a.listEventsHandler))
	r.Post("/events", a.wrap(a.createEventHandler))
	r.Get("/events/{eventID}", a.wrap(a.getEventHandler))
	r.Patch("/events/{eventID}", a.wrap(a.updateEventHandler))
	r.Delete("/events/{eventID}", a.wrap(a.deleteEventHandler))
	r.Post("/events/{eventID}/publish", a.wrap(a.publishEventHandler))
}

// listEventsHandler godoc
//
//	@Summary List events
//	@Tags events
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events [get]
func (a *API) listEventsHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
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

// createEventHandler godoc
//
//	@Summary Create an event
//	@Tags events
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body event.Input true "Event"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events [post]
func (a *API) createEventHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
	in, e := decode[event.Input](w, r)
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

// getEventHandler godoc
//
//	@Summary Get an event
//	@Tags events
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID} [get]
func (a *API) getEventHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
	v, e := s.Get(r.Context(), userID(r), batchID(r), param(r, "eventID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
	return send(w, 200, v)
}

// updateEventHandler godoc
//
//	@Summary Update an event
//	@Tags events
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Param payload body event.Input true "Event"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID} [patch]
func (a *API) updateEventHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
	in, e := decode[event.Input](w, r)
	if e != nil {
		return e
	}
	if e = s.Update(r.Context(), userID(r), batchID(r), param(r, "eventID"), in); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}

// deleteEventHandler godoc
//
//	@Summary Archive an event
//	@Tags events
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID} [delete]
func (a *API) deleteEventHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "eventID"), false); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}

// publishEventHandler godoc
//
//	@Summary Publish an event
//	@Tags events
//	@Param batchID path string true "Batch ID"
//	@Param eventID path string true "Event ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/events/{eventID}/publish [post]
func (a *API) publishEventHandler(w http.ResponseWriter, r *http.Request) error {
	s := event.Service{Pool: a.Pool}
	if e := s.Transition(r.Context(), userID(r), batchID(r), param(r, "eventID"), true); e != nil {
		return e
	}
	return send(w, 204, nil)
	return send(w, 204, nil)
}
