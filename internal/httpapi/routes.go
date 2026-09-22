package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/go-chi/chi/v5"
)

// Middleware applies the API-wide request, security, CORS, rate-limit, and
// logging behavior. Route composition lives in cmd/api.
func (a *API) Middleware(next http.Handler) http.Handler { return a.middleware(next) }

func (a *API) Authenticated(next http.Handler) http.Handler { return a.authenticated(next) }

func (a *API) AuthenticatedBasic(next http.Handler) http.Handler {
	return a.authenticatedBasic(next)
}

func (a *API) NotFoundHandler() http.HandlerFunc {
	return a.wrap(a.notFoundHandler)
}

func (a *API) notFoundHandler(http.ResponseWriter, *http.Request) error { return apperror.ErrNotFound }

func (a *API) MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = send(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]any{
			"code":       "method_not_allowed",
			"message":    "Method not allowed",
			"request_id": r.Context().Value(requestIDKey),
		}})
	}
}

func (a *API) LiveHandler() http.HandlerFunc {
	return a.wrap(a.liveHandler)
}

// liveHandler godoc
//
//	@Summary Check liveness
//	@Description Reports whether the process is running.
//	@Tags health
//	@Produce json
//	@Success 200 {object} map[string]string
//	@Security basicAuth
//	@Router /health/live [get]
func (a *API) liveHandler(w http.ResponseWriter, _ *http.Request) error {
	return send(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := a.Pool.Ping(ctx); err != nil {
			_ = send(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		_ = send(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func (a *API) SwaggerRoutes(r chi.Router)       { a.swaggerRoutes(r) }
func (a *API) AuthRoutes(r chi.Router)          { a.authRoutes(r) }
func (a *API) FileRoutes(r chi.Router)          { a.fileRoutes(r) }
func (a *API) GalleryPublicRoutes(r chi.Router) { a.galleryPublicRoutes(r) }
func (a *API) PublicBatchRoutes(r chi.Router)   { a.publicBatchRoutes(r) }
func (a *API) PublicImageRoutes(r chi.Router)   { a.publicImageRoutes(r) }
func (a *API) MeRoutes(r chi.Router)            { a.meRoutes(r) }
func (a *API) ProfileImageRoutes(r chi.Router)  { a.profileImageRoutes(r) }
func (a *API) NotificationRoutes(r chi.Router)  { a.notificationRoutes(r) }
func (a *API) AdminRoutes(r chi.Router)         { a.adminRoutes(r) }
func (a *API) GalleryAdminRoutes(r chi.Router)  { a.galleryAdminRoutes(r) }
func (a *API) BatchIndexRoutes(r chi.Router)    { a.batchIndexRoutes(r) }
func (a *API) BatchRoutes(r chi.Router)         { a.batchRoutes(r) }
func (a *API) SemesterRoutes(r chi.Router)      { a.semesterRoutes(r) }
func (a *API) ModuleRoutes(r chi.Router)        { a.moduleRoutes(r) }
func (a *API) AnnouncementRoutes(r chi.Router)  { a.announcementRoutes(r) }
func (a *API) ResourceRoutes(r chi.Router)      { a.resourceRoutes(r) }
func (a *API) LessonRoutes(r chi.Router)        { a.lessonRoutes(r) }
func (a *API) LinkRoutes(r chi.Router)          { a.linkRoutes(r) }
func (a *API) EventRoutes(r chi.Router)         { a.eventRoutes(r) }
func (a *API) EventImageRoutes(r chi.Router)    { a.eventImageRoutes(r) }
func (a *API) PositionRoutes(r chi.Router)      { a.positionRoutes(r) }
func (a *API) SupportRoutes(r chi.Router)       { a.supportRoutes(r) }
func (a *API) FundRoutes(r chi.Router)          { a.fundRoutes(r) }
func (a *API) AuditHandler() http.HandlerFunc   { return a.auditHandler() }
