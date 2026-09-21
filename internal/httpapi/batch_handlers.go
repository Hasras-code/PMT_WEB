package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/membership"
	"github.com/Hasras-code/PMT_WEB.git/internal/position"

	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *API) publicBatchRoutes(r chi.Router) {
	r.Get("/public/batches/{slug}", a.wrap(a.publicBatchHandler))
	r.Get("/public/batches/{slug}/events", a.wrap(a.publicBatchEventsHandler))
	r.Get("/public/batches/{slug}/events/{eventID}", a.wrap(a.publicBatchEventHandler))
	r.Get("/public/batches/{slug}/positions", a.wrap(a.publicBatchPositionsHandler))
}

// publicBatchHandler godoc
//
//	@Summary Get a public batch
//	@Tags batches
//	@Produce json
//	@Param slug path string true "Batch slug"
//	@Success 200 {object} map[string]any
//	@Router /v1/public/batches/{slug} [get]
func (a *API) publicBatchHandler(w http.ResponseWriter, r *http.Request) error {
	batches := batch.Service{Pool: a.Pool}
	value, err := batches.Public(r.Context(), param(r, "slug"))
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, value)
}

// publicBatchEventsHandler godoc
//
//	@Summary List public batch events
//	@Tags events
//	@Produce json
//	@Param slug path string true "Batch slug"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Router /v1/public/batches/{slug}/events [get]
func (a *API) publicBatchEventsHandler(w http.ResponseWriter, r *http.Request) error {
	batches := batch.Service{Pool: a.Pool}
	return a.publicBatchEvents(w, r, batches)
}

// publicBatchEventHandler godoc
//
//	@Summary Get a public batch event
//	@Tags events
//	@Produce json
//	@Param slug path string true "Batch slug"
//	@Param eventID path string true "Event ID"
//	@Success 200 {array} map[string]any
//	@Router /v1/public/batches/{slug}/events/{eventID} [get]
func (a *API) publicBatchEventHandler(w http.ResponseWriter, r *http.Request) error {
	batches := batch.Service{Pool: a.Pool}
	return a.publicBatchEvents(w, r, batches)
}

func (a *API) publicBatchEvents(w http.ResponseWriter, r *http.Request, batches batch.Service) error {
	limit, offset, err := page(r)
	if err != nil {
		return err
	}
	value, err := batches.PublicEvents(r.Context(), param(r, "slug"), param(r, "eventID"), limit, offset)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, value)
}

// publicBatchPositionsHandler godoc
//
//	@Summary List public batch positions
//	@Tags positions
//	@Produce json
//	@Param slug path string true "Batch slug"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Router /v1/public/batches/{slug}/positions [get]
func (a *API) publicBatchPositionsHandler(w http.ResponseWriter, r *http.Request) error {
	positions := position.Service{Pool: a.Pool}
	limit, offset, err := page(r)
	if err != nil {
		return err
	}
	value, err := positions.Public(r.Context(), param(r, "slug"), limit, offset)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, value)
}

func (a *API) batchIndexRoutes(r chi.Router) {
	r.Get("/batches", a.wrap(a.listBatchesHandler))
}

// listBatchesHandler godoc
//
//	@Summary List batches
//	@Tags batches
//	@Produce json
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches [get]
func (a *API) listBatchesHandler(w http.ResponseWriter, r *http.Request) error {
	s := batch.Service{Pool: a.Pool}
	limit, offset, err := page(r)
	if err != nil {
		return err
	}
	batches, err := s.List(r.Context(), userID(r), limit, offset)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, batches)
}

func (a *API) batchRoutes(r chi.Router) {
	r.Get("/", a.wrap(a.getBatchHandler))
	r.Patch("/", a.wrap(a.updateBatchHandler))
	r.Get("/profile", a.wrap(a.getBatchProfileHandler))
	r.Patch("/profile", a.wrap(a.updateBatchProfileHandler))
	r.Post("/profile/uploads", a.wrap(a.authorizeBatchProfileUploadHandler))
	r.Get("/members", a.wrap(a.listBatchMembersHandler))
	r.Get("/members/{membershipID}", a.wrap(a.getBatchMemberHandler))
	r.Post("/members", a.wrap(a.createBatchMemberHandler))
	r.Get("/roles", a.wrap(a.listBatchRolesHandler))
	r.Post("/members/{membershipID}/roles", a.wrap(a.setBatchMemberRoleHandler))
	r.Delete("/members/{membershipID}/roles/{roleCode}", a.wrap(a.deleteBatchMemberRoleHandler))
	r.Post("/members/{membershipID}/suspend", a.wrap(a.suspendBatchMemberHandler))
	r.Post("/members/{membershipID}/reactivate", a.wrap(a.reactivateBatchMemberHandler))
	r.Post("/members/{membershipID}/graduate", a.wrap(a.graduateBatchMemberHandler))
	r.Post("/members/{membershipID}/leave", a.wrap(a.leaveBatchMemberHandler))
}

// getBatchHandler godoc
//
//	@Summary Get the current batch
//	@Tags batches
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID} [get]
func (a *API) getBatchHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (batch.Service{Pool: a.Pool}).Get(r.Context(), userID(r), batchID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateBatchHandler godoc
//
//	@Summary Update the current batch
//	@Tags batches
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body map[string]string true "Batch fields"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID} [patch]
func (a *API) updateBatchHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}](w, r)
	if e != nil {
		return e
	}
	if e = (batch.Service{Pool: a.Pool}).Update(r.Context(), userID(r), batchID(r), in.Name, in.Description); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// getBatchProfileHandler godoc
//
//	@Summary Get the batch profile
//	@Tags batches
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/profile [get]
func (a *API) getBatchProfileHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (batch.Service{Pool: a.Pool}).Profile(r.Context(), userID(r), batchID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateBatchProfileHandler godoc
//
//	@Summary Update the batch profile
//	@Tags batches
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body batch.Profile true "Batch profile"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/profile [patch]
func (a *API) updateBatchProfileHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[batch.Profile](w, r)
	if e != nil {
		return e
	}
	if e = (batch.Service{Pool: a.Pool}).SetProfile(r.Context(), userID(r), batchID(r), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// listBatchMembersHandler godoc
//
//	@Summary List batch members
//	@Tags memberships
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members [get]
func (a *API) listBatchMembersHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listBatchMembers(w, r, "")
}

// getBatchMemberHandler godoc
//
//	@Summary Get a batch member
//	@Tags memberships
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID} [get]
func (a *API) getBatchMemberHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listBatchMembers(w, r, param(r, "membershipID"))
}

func (a *API) listBatchMembers(w http.ResponseWriter, r *http.Request, membershipID string) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (membership.Service{Pool: a.Pool}).List(r.Context(), userID(r), batchID(r), membershipID, l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// createBatchMemberHandler godoc
//
//	@Summary Add a batch member
//	@Tags memberships
//	@Accept json
//	@Param batchID path string true "Batch ID"
//	@Param payload body map[string]string true "User"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members [post]
func (a *API) createBatchMemberHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		UserID string `json:"user_id"`
	}](w, r)
	if e != nil {
		return e
	}
	id, e := (membership.Service{Pool: a.Pool}).Create(r.Context(), userID(r), batchID(r), in.UserID, false)
	if e != nil {
		return e
	}
	return created(w, id)
}

// listBatchRolesHandler godoc
//
//	@Summary List batch roles
//	@Tags memberships
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/roles [get]
func (a *API) listBatchRolesHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (membership.Service{Pool: a.Pool}).Catalog(r.Context(), userID(r), batchID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// setBatchMemberRoleHandler godoc
//
//	@Summary Set a batch member role
//	@Tags memberships
//	@Accept json
//	@Produce json
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Param payload body map[string]string true "Role"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID}/roles [post]
func (a *API) setBatchMemberRoleHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		Role string `json:"role"`
	}](w, r)
	if e != nil {
		return e
	}
	s := membership.Service{Pool: a.Pool}
	if e = s.Role(r.Context(), userID(r), batchID(r), param(r, "membershipID"), in.Role, false, false); e != nil {
		return e
	}
	v, e := s.List(r.Context(), userID(r), batchID(r), param(r, "membershipID"), 1, 0)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// deleteBatchMemberRoleHandler godoc
//
//	@Summary Remove a batch member role
//	@Tags memberships
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Param roleCode path string true "Role code"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID}/roles/{roleCode} [delete]
func (a *API) deleteBatchMemberRoleHandler(w http.ResponseWriter, r *http.Request) error {
	if e := (membership.Service{Pool: a.Pool}).Role(r.Context(), userID(r), batchID(r), param(r, "membershipID"), param(r, "roleCode"), true, false); e != nil {
		return e
	}
	return send(w, 204, nil)
}

func (a *API) setBatchMemberStatusHandler(w http.ResponseWriter, r *http.Request, status string) error {
	if e := (membership.Service{Pool: a.Pool}).SetStatus(r.Context(), userID(r), batchID(r), param(r, "membershipID"), status); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// suspendBatchMemberHandler godoc
//
//	@Summary Suspend a batch member
//	@Tags memberships
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID}/suspend [post]
func (a *API) suspendBatchMemberHandler(w http.ResponseWriter, r *http.Request) error {
	return a.setBatchMemberStatusHandler(w, r, "SUSPENDED")
}

// reactivateBatchMemberHandler godoc
//
//	@Summary Reactivate a batch member
//	@Tags memberships
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID}/reactivate [post]
func (a *API) reactivateBatchMemberHandler(w http.ResponseWriter, r *http.Request) error {
	return a.setBatchMemberStatusHandler(w, r, "ACTIVE")
}

// graduateBatchMemberHandler godoc
//
//	@Summary Graduate a batch member
//	@Tags memberships
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID}/graduate [post]
func (a *API) graduateBatchMemberHandler(w http.ResponseWriter, r *http.Request) error {
	return a.setBatchMemberStatusHandler(w, r, "GRADUATED")
}

// leaveBatchMemberHandler godoc
//
//	@Summary Mark a batch member as left
//	@Tags memberships
//	@Param batchID path string true "Batch ID"
//	@Param membershipID path string true "Membership ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/batches/{batchID}/members/{membershipID}/leave [post]
func (a *API) leaveBatchMemberHandler(w http.ResponseWriter, r *http.Request) error {
	return a.setBatchMemberStatusHandler(w, r, "LEFT")
}
