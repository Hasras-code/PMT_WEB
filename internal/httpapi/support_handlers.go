package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/complaint"
	"github.com/Hasras-code/PMT_WEB.git/internal/feedback"
	"github.com/go-chi/chi/v5"
)

func (a *API) supportRoutes(r chi.Router) {
	r.Post("/complaints", a.wrap(a.createComplaintHandler))
	r.Get("/complaints", a.wrap(a.listComplaintsHandler))
	r.Get("/complaints/mine", a.wrap(a.listMyComplaintsHandler))
	r.Get("/complaints/{complaintID}", a.wrap(a.getComplaintHandler))
	r.Get("/complaints/{complaintID}/messages", a.wrap(a.listComplaintMessagesHandler))
	r.Post("/complaints/{complaintID}/messages", a.wrap(a.replyComplaintHandler))
	r.Patch("/complaints/{complaintID}/status", a.wrap(a.updateComplaintStatusHandler))
	r.Patch("/complaints/{complaintID}/assignee", a.wrap(a.assignComplaintHandler))
	r.Post("/feedback", a.wrap(a.createFeedbackHandler))
	r.Get("/feedback", a.wrap(a.listFeedbackHandler))
	r.Patch("/feedback/{feedbackID}/status", a.wrap(a.updateFeedbackStatusHandler))
}

// createComplaintHandler godoc
// @Summary Create a complaint
// @Tags support
// @Accept json
// @Param payload body complaint.Input true "Complaint"
// @Success 201 {object} map[string]string
// @Security bearerAuth
// @Router /v1/complaints [post]
func (a *API) createComplaintHandler(w http.ResponseWriter, r *http.Request) error {
	s := complaint.Service{Pool: a.Pool}
	in, e := decode[complaint.Input](w, r)
	if e != nil {
		return e
	}
	id, e := s.Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// listComplaintsHandler godoc
// @Summary List complaints
// @Tags support
// @Produce json
// @Param batchID path string true "Batch ID"
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/complaints [get]
func (a *API) listComplaintsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listComplaints(w, r, false)
}

// listMyComplaintsHandler godoc
// @Summary List my complaints
// @Tags support
// @Produce json
// @Param batchID path string true "Batch ID"
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/complaints/mine [get]
func (a *API) listMyComplaintsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listComplaints(w, r, true)
}
func (a *API) listComplaints(w http.ResponseWriter, r *http.Request, mine bool) error {
	s := complaint.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.List(r.Context(), userID(r), batchID(r), mine, l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// getComplaintHandler godoc
// @Summary Get a complaint
// @Tags support
// @Produce json
// @Param complaintID path string true "Complaint ID"
// @Success 200 {object} map[string]any
// @Security bearerAuth
// @Router /v1/complaints/{complaintID} [get]
func (a *API) getComplaintHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (complaint.Service{Pool: a.Pool}).Get(r.Context(), userID(r), batchID(r), param(r, "complaintID"))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// listComplaintMessagesHandler godoc
// @Summary List complaint messages
// @Tags support
// @Produce json
// @Param complaintID path string true "Complaint ID"
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/complaints/{complaintID}/messages [get]
func (a *API) listComplaintMessagesHandler(w http.ResponseWriter, r *http.Request) error {
	s := complaint.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.Messages(r.Context(), userID(r), batchID(r), param(r, "complaintID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// replyComplaintHandler godoc
// @Summary Reply to a complaint
// @Tags support
// @Accept json
// @Param complaintID path string true "Complaint ID"
// @Param payload body map[string]string true "Message"
// @Success 204
// @Security bearerAuth
// @Router /v1/complaints/{complaintID}/messages [post]
func (a *API) replyComplaintHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		Message string `json:"message"`
	}](w, r)
	if e != nil {
		return e
	}
	if e = (complaint.Service{Pool: a.Pool}).Reply(r.Context(), userID(r), batchID(r), param(r, "complaintID"), in.Message); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// updateComplaintStatusHandler godoc
// @Summary Update complaint status
// @Tags support
// @Accept json
// @Param complaintID path string true "Complaint ID"
// @Param payload body map[string]string true "Status"
// @Success 204
// @Security bearerAuth
// @Router /v1/complaints/{complaintID}/status [patch]
func (a *API) updateComplaintStatusHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		Status string `json:"status"`
	}](w, r)
	if e != nil {
		return e
	}
	if e = (complaint.Service{Pool: a.Pool}).Transition(r.Context(), userID(r), batchID(r), param(r, "complaintID"), in.Status); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// assignComplaintHandler godoc
// @Summary Assign a complaint
// @Tags support
// @Accept json
// @Param complaintID path string true "Complaint ID"
// @Param payload body map[string]string true "Assignee"
// @Success 204
// @Security bearerAuth
// @Router /v1/complaints/{complaintID}/assignee [patch]
func (a *API) assignComplaintHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		UserID string `json:"user_id"`
	}](w, r)
	if e != nil {
		return e
	}
	if e = (complaint.Service{Pool: a.Pool}).Assign(r.Context(), userID(r), batchID(r), param(r, "complaintID"), in.UserID); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// createFeedbackHandler godoc
// @Summary Create feedback
// @Tags support
// @Accept json
// @Param payload body feedback.Input true "Feedback"
// @Success 201 {object} map[string]string
// @Security bearerAuth
// @Router /v1/feedback [post]
func (a *API) createFeedbackHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[feedback.Input](w, r)
	if e != nil {
		return e
	}
	id, e := (feedback.Service{Pool: a.Pool}).Create(r.Context(), userID(r), batchID(r), in)
	if e != nil {
		return e
	}
	return created(w, id)
}

// listFeedbackHandler godoc
// @Summary List feedback
// @Tags support
// @Produce json
// @Success 200 {array} map[string]any
// @Security bearerAuth
// @Router /v1/feedback [get]
func (a *API) listFeedbackHandler(w http.ResponseWriter, r *http.Request) error {
	s := feedback.Service{Pool: a.Pool}
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

// updateFeedbackStatusHandler godoc
// @Summary Update feedback status
// @Tags support
// @Accept json
// @Param feedbackID path string true "Feedback ID"
// @Param payload body map[string]string true "Status"
// @Success 204
// @Security bearerAuth
// @Router /v1/feedback/{feedbackID}/status [patch]
func (a *API) updateFeedbackStatusHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		Status string `json:"status"`
	}](w, r)
	if e != nil {
		return e
	}
	if e = (feedback.Service{Pool: a.Pool}).Status(r.Context(), userID(r), batchID(r), param(r, "feedbackID"), in.Status); e != nil {
		return e
	}
	return send(w, 204, nil)
}
