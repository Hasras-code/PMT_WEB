package main

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/httpapi"
	"github.com/go-chi/chi/v5"
)

// mount is the single composition point for the HTTP transport. Feature
// handlers remain in internal/httpapi; this method owns their URL hierarchy
// and middleware boundaries.
func (a *app) mount() *chi.Mux {
	api := &httpapi.API{
		Pool:    a.pool,
		Auth:    a.auth,
		Files:   a.files,
		Uploads: a.uploads,
		Config:  a.cfg,
		Log:     a.logger,
	}

	r := chi.NewRouter()
	r.Use(api.Middleware)
	if a.cfg.Env != "production" {
		api.SwaggerRoutes(r)
	}
	r.NotFound(api.NotFoundHandler())
	r.MethodNotAllowed(api.MethodNotAllowedHandler())
	r.With(api.AuthenticatedBasic).Get("/health/live", api.LiveHandler())
	r.With(api.AuthenticatedBasic).Get("/health/ready", api.ReadyHandler())

	r.Route("/v1", func(r chi.Router) {
		api.AuthRoutes(r)
		api.FileRoutes(r)
		api.GalleryPublicRoutes(r)
		api.PublicBatchRoutes(r)
		api.PublicImageRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(api.Authenticated)
			api.MeRoutes(r)
			api.ProfileImageRoutes(r)
			api.NotificationRoutes(r)
			api.AdminRoutes(r)
			api.GalleryAdminRoutes(r)
			api.BatchIndexRoutes(r)

			r.Route("/batches/{batchID}", func(r chi.Router) {
				api.BatchRoutes(r)
				api.SemesterRoutes(r)
				api.ModuleRoutes(r)
				api.AnnouncementRoutes(r)
				api.ResourceRoutes(r)
				api.LessonRoutes(r)
				api.LinkRoutes(r)
				api.EventRoutes(r)
				api.EventImageRoutes(r)
				api.PositionRoutes(r)
				api.SupportRoutes(r)
				api.FundRoutes(r)
				r.Get("/audit-logs", api.AuditHandler())
			})
		})
	})

	return r
}
