package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/batch"
	"github.com/Hasras-code/PMT_WEB.git/internal/notification"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
)

func (a *API) adminRoutes(r chi.Router) {
	r.Post("/admin/batches", a.wrap(a.adminCreateBatchHandler))
	r.Post("/admin/batches/{batchID}/archive", a.wrap(a.adminArchiveBatchHandler))
	r.Get("/admin/users", a.wrap(a.adminListUsersHandler))
	r.Get("/admin/users/{userID}", a.wrap(a.adminListUserHandler))
	r.Get("/admin/stats", a.wrap(a.adminStatsHandler))
	r.Get("/admin/roles", a.wrap(a.adminListRolesHandler))
	r.Get("/admin/batches", a.wrap(a.adminListBatchesHandler))
	r.Post("/admin/users/{userID}/roles", a.wrap(a.adminSetUserRoleHandler))
	r.Delete("/admin/users/{userID}/roles/{roleCode}", a.wrap(a.adminDeleteUserRoleHandler))
	r.Post("/admin/users/{userID}/suspend", a.wrap(a.adminSuspendUserHandler))
	r.Post("/admin/users/{userID}/reactivate", a.wrap(a.adminReactivateUserHandler))
	r.Post("/admin/users/{userID}/archive", a.wrap(a.adminArchiveUserHandler))
	r.Get("/admin/audit-logs", a.auditHandler())
}

// adminCreateBatchHandler godoc
//
//	@Summary Create a batch
//	@Tags admin
//	@Accept json
//	@Param payload body batch.Input true "Batch"
//	@Success 201 {object} map[string]string
//	@Security bearerAuth
//	@Router /v1/admin/batches [post]
func (a *API) adminCreateBatchHandler(w http.ResponseWriter, r *http.Request) error {
	b := batch.Service{Pool: a.Pool}
	in, e := decode[batch.Input](w, r)
	if e != nil {
		return e
	}
	id, e := b.Create(r.Context(), userID(r), in, false)
	if e != nil {
		return e
	}
	return created(w, id)
}

// adminArchiveBatchHandler godoc
//
//	@Summary Archive a batch
//	@Tags admin
//	@Param batchID path string true "Batch ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/batches/{batchID}/archive [post]
func (a *API) adminArchiveBatchHandler(w http.ResponseWriter, r *http.Request) error {
	b := batch.Service{Pool: a.Pool}
	if e := b.Archive(r.Context(), userID(r), batchID(r)); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// adminListUsersHandler godoc
//
//	@Summary List users
//	@Tags admin
//	@Produce json
//	@Param userID path string false "User ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/admin/users [get]
func (a *API) adminListUsersHandler(w http.ResponseWriter, r *http.Request) error {
	u := user.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := u.AdminList(r.Context(), userID(r), param(r, "userID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// adminStatsHandler godoc
//
//	@Summary Get platform administration statistics
//	@Tags admin
//	@Produce json
//	@Success 200 {object} map[string]int
//	@Security bearerAuth
//	@Router /v1/admin/stats [get]
func (a *API) adminStatsHandler(w http.ResponseWriter, r *http.Request) error {
	u := user.Service{Pool: a.Pool}
	v, e := u.AdminStats(r.Context(), userID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// adminListUserHandler godoc
//
//	@Summary Get a user's admin view
//	@Tags admin
//	@Produce json
//	@Param userID path string true "User ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/admin/users/{userID} [get]
func (a *API) adminListUserHandler(w http.ResponseWriter, r *http.Request) error {
	u := user.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := u.AdminList(r.Context(), userID(r), param(r, "userID"), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

func (a *API) adminListRolesHandler(w http.ResponseWriter, r *http.Request) error {
	u := user.Service{Pool: a.Pool}
	v, e := u.PlatformRoles(r.Context(), userID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

func (a *API) adminListBatchesHandler(w http.ResponseWriter, r *http.Request) error {
	u := user.Service{Pool: a.Pool}
	v, e := u.AdminBatches(r.Context(), userID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

func (a *API) adminSetUserRoleHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[struct {
		Role    string `json:"role"`
		BatchID string `json:"batch_id"`
	}](w, r)
	if e != nil {
		return e
	}
	u := user.Service{Pool: a.Pool}
	if e = u.Role(r.Context(), userID(r), param(r, "userID"), in.Role, in.BatchID, false); e != nil {
		return e
	}
	return send(w, 204, nil)
}

func (a *API) adminDeleteUserRoleHandler(w http.ResponseWriter, r *http.Request) error {
	batchID := r.URL.Query().Get("batch_id")
	u := user.Service{Pool: a.Pool}
	if e := u.Role(r.Context(), userID(r), param(r, "userID"), param(r, "roleCode"), batchID, true); e != nil {
		return e
	}
	return send(w, 204, nil)
}

func (a *API) adminSetUserStatusHandler(w http.ResponseWriter, r *http.Request, status string) error {
	u := user.Service{Pool: a.Pool}
	if e := u.Status(r.Context(), userID(r), param(r, "userID"), status); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// adminSuspendUserHandler godoc
//
//	@Summary Suspend a user
//	@Tags admin
//	@Param userID path string true "User ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/users/{userID}/suspend [post]
func (a *API) adminSuspendUserHandler(w http.ResponseWriter, r *http.Request) error {
	return a.adminSetUserStatusHandler(w, r, "SUSPENDED")
}

// adminReactivateUserHandler godoc
//
//	@Summary Reactivate a user
//	@Tags admin
//	@Param userID path string true "User ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/users/{userID}/reactivate [post]
func (a *API) adminReactivateUserHandler(w http.ResponseWriter, r *http.Request) error {
	return a.adminSetUserStatusHandler(w, r, "ACTIVE")
}

// adminArchiveUserHandler godoc
//
//	@Summary Archive a user
//	@Tags admin
//	@Param userID path string true "User ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/admin/users/{userID}/archive [post]
func (a *API) adminArchiveUserHandler(w http.ResponseWriter, r *http.Request) error {
	return a.adminSetUserStatusHandler(w, r, "ARCHIVED")
}
func (a *API) auditHandler() http.HandlerFunc {
	return a.wrap(a.auditLogsHandler)
}

// auditLogsHandler godoc
//
//	@Summary List audit logs
//	@Tags admin
//	@Produce json
//	@Param batchID path string false "Batch ID"
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/admin/audit-logs [get]
func (a *API) auditLogsHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := audit.List(r.Context(), a.Pool, userID(r), batchID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
func (a *API) notificationRoutes(r chi.Router) {
	r.Get("/me/notifications", a.wrap(a.listNotificationsHandler))
	r.Patch("/me/notifications/{notificationID}", a.wrap(a.readNotificationHandler))
	r.Post("/me/notifications/read-all", a.wrap(a.readAllNotificationsHandler))
}

// listNotificationsHandler godoc
//
//	@Summary List notifications
//	@Tags notifications
//	@Produce json
//	@Param limit query int false "Limit"
//	@Param offset query int false "Offset"
//	@Success 200 {array} map[string]any
//	@Security bearerAuth
//	@Router /v1/me/notifications [get]
func (a *API) listNotificationsHandler(w http.ResponseWriter, r *http.Request) error {
	s := notification.Service{Pool: a.Pool}
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := s.List(r.Context(), userID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// readNotificationHandler godoc
//
//	@Summary Read a notification
//	@Tags notifications
//	@Param notificationID path string true "Notification ID"
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/me/notifications/{notificationID} [patch]
func (a *API) readNotificationHandler(w http.ResponseWriter, r *http.Request) error {
	s := notification.Service{Pool: a.Pool}
	if e := s.Read(r.Context(), userID(r), param(r, "notificationID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// readAllNotificationsHandler godoc
//
//	@Summary Read all notifications
//	@Tags notifications
//	@Success 204
//	@Security bearerAuth
//	@Router /v1/me/notifications/read-all [post]
func (a *API) readAllNotificationsHandler(w http.ResponseWriter, r *http.Request) error {
	s := notification.Service{Pool: a.Pool}
	if e := s.Read(r.Context(), userID(r), ""); e != nil {
		return e
	}
	return send(w, 204, nil)
}
