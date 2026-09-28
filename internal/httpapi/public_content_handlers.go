package httpapi

import (
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/publiccontent"
	"github.com/go-chi/chi/v5"
)

func (a *API) publicContentRoutes(r chi.Router) {
	r.Get("/public/hero-slides", a.wrap(a.publicHeroSlidesHandler))
	r.Get("/public/achievements", a.wrap(a.publicAchievementsHandler))
	r.Get("/public/events", a.wrap(a.publicEventsHandler))
	r.Get("/public/home-gallery", a.wrap(a.publicHomeGalleryHandler))
	r.Get("/public/featured-reps", a.wrap(a.publicFeaturedRepsHandler))
	r.Get("/public/socials", a.wrap(a.publicSocialsHandler))
}

func (a *API) publicContentAdminRoutes(r chi.Router) {
	r.Get("/admin/public/hero-slides", a.wrap(a.adminHeroSlidesHandler))
	r.Post("/admin/public/hero-slides", a.wrap(a.createHeroSlideHandler))
	r.Delete("/admin/public/hero-slides/{heroSlideID}", a.wrap(a.deleteHeroSlideHandler))
	r.Get("/admin/public/events", a.wrap(a.adminPublicEventsHandler))
	r.Post("/admin/public/events", a.wrap(a.createPublicEventHandler))
	r.Delete("/admin/public/events/{publicEventID}", a.wrap(a.deletePublicEventHandler))
	r.Get("/admin/public/gallery", a.wrap(a.adminPublicGalleryHandler))
	r.Post("/admin/public/gallery", a.wrap(a.createPublicGalleryHandler))
	r.Delete("/admin/public/gallery/{publicGalleryID}", a.wrap(a.deletePublicGalleryHandler))
	r.Get("/admin/public/achievements", a.wrap(a.adminAchievementsHandler))
	r.Post("/admin/public/achievements", a.wrap(a.createAchievementHandler))
	r.Delete("/admin/public/achievements/{achievementID}", a.wrap(a.deleteAchievementHandler))
	r.Get("/admin/public/reps", a.wrap(a.adminPublicRepsHandler))
	r.Post("/admin/public/reps/{assignmentID}/toggle-featured", a.wrap(a.toggleFeaturedRepHandler))
	r.Get("/admin/public/socials", a.wrap(a.adminSocialsHandler))
	r.Post("/admin/public/socials", a.wrap(a.saveSocialsHandler))
}

func (a *API) publicContentService() publiccontent.Service {
	return publiccontent.Service{Pool: a.Pool}
}

func contentPage(r *http.Request) (int, int, error) { return page(r) }

func (a *API) publicHeroSlidesHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, err := contentPage(r)
	if err != nil {
		return err
	}
	v, err := a.publicContentService().Heroes(r.Context(), "", l, o)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) adminHeroSlidesHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, err := contentPage(r)
	if err != nil {
		return err
	}
	v, err := a.publicContentService().Heroes(r.Context(), userID(r), l, o)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) createHeroSlideHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[publiccontent.HeroInput](w, r)
	if err != nil {
		return err
	}
	id, err := a.publicContentService().SaveHero(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	return created(w, id)
}

func (a *API) deleteHeroSlideHandler(w http.ResponseWriter, r *http.Request) error {
	err := a.publicContentService().Delete(r.Context(), userID(r), "public_hero_slides", "public_hero_slide", "HERO_SLIDE_DELETED", param(r, "heroSlideID"))
	if err != nil {
		return err
	}
	return send(w, http.StatusNoContent, nil)
}

func (a *API) publicEventsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPublicEvents(w, r, "")
}

func (a *API) adminPublicEventsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPublicEvents(w, r, userID(r))
}

func (a *API) listPublicEvents(w http.ResponseWriter, r *http.Request, user string) error {
	l, o, err := contentPage(r)
	if err != nil {
		return err
	}
	v, err := a.publicContentService().Events(r.Context(), user, l, o)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) createPublicEventHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[publiccontent.EventInput](w, r)
	if err != nil {
		return err
	}
	id, err := a.publicContentService().SaveEvent(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	return created(w, id)
}

func (a *API) deletePublicEventHandler(w http.ResponseWriter, r *http.Request) error {
	err := a.publicContentService().Delete(r.Context(), userID(r), "public_events", "public_event", "PUBLIC_EVENT_DELETED", param(r, "publicEventID"))
	if err != nil {
		return err
	}
	return send(w, http.StatusNoContent, nil)
}

func (a *API) publicHomeGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPublicContentGallery(w, r, "", true)
}

func (a *API) adminPublicGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPublicContentGallery(w, r, userID(r), false)
}

func (a *API) listPublicContentGallery(w http.ResponseWriter, r *http.Request, user string, publicOnly bool) error {
	l, o, err := contentPage(r)
	if err != nil {
		return err
	}
	v, err := a.publicContentService().Gallery(r.Context(), user, publicOnly, l, o)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) createPublicGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[publiccontent.GalleryInput](w, r)
	if err != nil {
		return err
	}
	id, err := a.publicContentService().SaveGallery(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	return created(w, id)
}

func (a *API) deletePublicGalleryHandler(w http.ResponseWriter, r *http.Request) error {
	err := a.publicContentService().Delete(r.Context(), userID(r), "public_gallery_items", "public_gallery_item", "PUBLIC_GALLERY_ITEM_DELETED", param(r, "publicGalleryID"))
	if err != nil {
		return err
	}
	return send(w, http.StatusNoContent, nil)
}

func (a *API) publicAchievementsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listAchievements(w, r, "")
}

func (a *API) adminAchievementsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listAchievements(w, r, userID(r))
}

func (a *API) listAchievements(w http.ResponseWriter, r *http.Request, user string) error {
	l, o, err := contentPage(r)
	if err != nil {
		return err
	}
	v, err := a.publicContentService().Achievements(r.Context(), user, l, o)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) createAchievementHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[publiccontent.AchievementInput](w, r)
	if err != nil {
		return err
	}
	id, err := a.publicContentService().SaveAchievement(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	return created(w, id)
}

func (a *API) deleteAchievementHandler(w http.ResponseWriter, r *http.Request) error {
	err := a.publicContentService().Delete(r.Context(), userID(r), "public_achievements", "public_achievement", "PUBLIC_ACHIEVEMENT_DELETED", param(r, "achievementID"))
	if err != nil {
		return err
	}
	return send(w, http.StatusNoContent, nil)
}

func (a *API) publicFeaturedRepsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPublicReps(w, r, "", true)
}

func (a *API) adminPublicRepsHandler(w http.ResponseWriter, r *http.Request) error {
	return a.listPublicReps(w, r, userID(r), false)
}

func (a *API) listPublicReps(w http.ResponseWriter, r *http.Request, user string, featuredOnly bool) error {
	l, o, err := contentPage(r)
	if err != nil {
		return err
	}
	v, err := a.publicContentService().Reps(r.Context(), user, featuredOnly, l, o)
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) toggleFeaturedRepHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[publiccontent.FeaturedInput](w, r)
	if err != nil {
		return err
	}
	if err = a.publicContentService().SetFeatured(r.Context(), userID(r), param(r, "assignmentID"), in.Featured); err != nil {
		return err
	}
	return send(w, http.StatusNoContent, nil)
}

func (a *API) publicSocialsHandler(w http.ResponseWriter, r *http.Request) error {
	v, err := a.publicContentService().Socials(r.Context(), "")
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) adminSocialsHandler(w http.ResponseWriter, r *http.Request) error {
	v, err := a.publicContentService().Socials(r.Context(), userID(r))
	if err != nil {
		return err
	}
	return send(w, http.StatusOK, v)
}

func (a *API) saveSocialsHandler(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[publiccontent.SocialInput](w, r)
	if err != nil {
		return err
	}
	if err = a.publicContentService().SaveSocials(r.Context(), userID(r), in); err != nil {
		return err
	}
	return send(w, http.StatusNoContent, nil)
}
