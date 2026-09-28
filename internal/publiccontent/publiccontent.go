package publiccontent

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	platformPermission    = "public_content.manage"
	contributorPermission = "public_content.contribute"
)

type Service struct{ Pool *pgxpool.Pool }

type HeroInput struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	CTAText  string `json:"cta_text"`
	ImageURL string `json:"image_url"`
	Priority int    `json:"priority"`
}

type EventInput struct {
	Title    string `json:"title"`
	Date     string `json:"date"`
	ImageURL string `json:"image_url"`
	Status   string `json:"status"`
	URL      string `json:"url"`
}

type GalleryInput struct {
	Title      string `json:"title"`
	Category   string `json:"category"`
	ImageURL   string `json:"image_url"`
	Visibility string `json:"visibility"`
}

type AchievementInput struct {
	Title string `json:"title"`
	Count int    `json:"count"`
	Icon  string `json:"icon"`
}

type SocialInput struct {
	WhatsApp  string `json:"whatsapp"`
	Facebook  string `json:"facebook"`
	Instagram string `json:"instagram"`
	YouTube   string `json:"youtube"`
}

type FeaturedInput struct {
	Featured bool `json:"isFeaturedOnHome"`
}

func clean(value string, max int, required bool) (string, error) {
	value = strings.TrimSpace(value)
	if (required && value == "") || len(value) > max {
		return "", apperror.ErrInvalid
	}
	return value, nil
}

func webURL(value string, required bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" && !required {
		return "", nil
	}
	if value == "" || len(value) > 2048 {
		return "", apperror.ErrInvalid
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return "", apperror.ErrInvalid
	}
	return u.String(), nil
}

func (s Service) requirePlatform(ctx context.Context, q authorization.Querier, user string) error {
	return authorization.RequirePlatform(ctx, q, user, platformPermission)
}

func (s Service) requireContributor(ctx context.Context, q authorization.Querier, user string) error {
	platform, err := authorization.Platform(ctx, q, user, platformPermission)
	if err != nil || platform {
		return err
	}
	var allowed bool
	err = q.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM batch_memberships m
		JOIN users u ON u.id=m.user_id
		JOIN batches b ON b.id=m.batch_id
		JOIN membership_roles mr ON mr.membership_id=m.id
		JOIN role_permissions rp ON rp.role_id=mr.role_id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE m.user_id=$1 AND m.status='ACTIVE' AND u.status='ACTIVE' AND b.status<>'ARCHIVED'
		  AND p.code=$2 AND p.scope='BATCH'
	)`, user, contributorPermission).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return apperror.ErrForbidden
	}
	return nil
}

func (s Service) list(ctx context.Context, user string, platformOnly bool, query string, args ...any) (json.RawMessage, error) {
	if user != "" {
		var err error
		if platformOnly {
			err = s.requirePlatform(ctx, s.Pool, user)
		} else {
			err = s.requireContributor(ctx, s.Pool, user)
		}
		if err != nil {
			return nil, err
		}
	}
	return db.JSON(s.Pool.QueryRow(ctx, query, args...))
}

func (s Service) Heroes(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return s.list(ctx, user, false, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,title,subtitle,cta_text,image_url,priority FROM public_hero_slides WHERE deleted_at IS NULL ORDER BY priority,created_at DESC,id LIMIT $1 OFFSET $2)t`, limit, offset)
}

func (s Service) SaveHero(ctx context.Context, user string, in HeroInput) (string, error) {
	var err error
	if in.Title, err = clean(in.Title, 200, true); err != nil {
		return "", err
	}
	if in.Subtitle, err = clean(in.Subtitle, 500, false); err != nil {
		return "", err
	}
	if in.CTAText, err = clean(in.CTAText, 100, false); err != nil || in.Priority < -10000 || in.Priority > 10000 {
		return "", apperror.ErrInvalid
	}
	if in.ImageURL, err = webURL(in.ImageURL, true); err != nil {
		return "", err
	}
	return s.insert(ctx, user, false, "HERO_SLIDE_CREATED", "public_hero_slide", func(tx pgx.Tx, id *string) error {
		return tx.QueryRow(ctx, `INSERT INTO public_hero_slides(title,subtitle,cta_text,image_url,priority,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, in.Title, in.Subtitle, in.CTAText, in.ImageURL, in.Priority, user).Scan(id)
	})
}

func (s Service) Events(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return s.list(ctx, user, false, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,title,event_date::text AS date,image_url,CASE WHEN event_date>=CURRENT_DATE THEN 'Upcoming' ELSE 'Past' END AS status,external_url AS url FROM public_events WHERE deleted_at IS NULL ORDER BY event_date DESC,id DESC LIMIT $1 OFFSET $2)t`, limit, offset)
}

func (s Service) SaveEvent(ctx context.Context, user string, in EventInput) (string, error) {
	var err error
	if in.Title, err = clean(in.Title, 300, true); err != nil {
		return "", err
	}
	if _, err = time.Parse("2006-01-02", in.Date); err != nil {
		return "", apperror.ErrInvalid
	}
	if in.Status != "" && in.Status != "Upcoming" && in.Status != "Past" {
		return "", apperror.ErrInvalid
	}
	if in.ImageURL, err = webURL(in.ImageURL, true); err != nil {
		return "", err
	}
	if in.URL, err = webURL(in.URL, false); err != nil {
		return "", err
	}
	return s.insert(ctx, user, false, "PUBLIC_EVENT_CREATED", "public_event", func(tx pgx.Tx, id *string) error {
		return tx.QueryRow(ctx, `INSERT INTO public_events(title,event_date,image_url,external_url,created_by) VALUES($1,$2::date,$3,$4,$5) RETURNING id`, in.Title, in.Date, in.ImageURL, in.URL, user).Scan(id)
	})
}

func (s Service) Gallery(ctx context.Context, user string, publicOnly bool, limit, offset int) (json.RawMessage, error) {
	if publicOnly {
		return s.list(ctx, "", false, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,title AS event_name,title,category,image_url,visibility FROM public_gallery_items WHERE deleted_at IS NULL AND visibility='Public' ORDER BY created_at DESC,id DESC LIMIT $1 OFFSET $2)t`, limit, offset)
	}
	return s.list(ctx, user, false, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,title,category,image_url,visibility FROM public_gallery_items WHERE deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT $1 OFFSET $2)t`, limit, offset)
}

func (s Service) SaveGallery(ctx context.Context, user string, in GalleryInput) (string, error) {
	var err error
	if in.Title, err = clean(in.Title, 300, true); err != nil {
		return "", err
	}
	if in.Category != "Academic" && in.Category != "Workshops" && in.Category != "Social" && in.Category != "Sports" {
		return "", apperror.ErrInvalid
	}
	if in.Visibility != "Public" && in.Visibility != "Internal" {
		return "", apperror.ErrInvalid
	}
	if in.ImageURL, err = webURL(in.ImageURL, true); err != nil {
		return "", err
	}
	return s.insert(ctx, user, false, "PUBLIC_GALLERY_ITEM_CREATED", "public_gallery_item", func(tx pgx.Tx, id *string) error {
		return tx.QueryRow(ctx, `INSERT INTO public_gallery_items(title,category,image_url,visibility,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id`, in.Title, in.Category, in.ImageURL, in.Visibility, user).Scan(id)
	})
}

func (s Service) Achievements(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return s.list(ctx, user, true, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,title,count,icon FROM public_achievements WHERE deleted_at IS NULL ORDER BY created_at,id LIMIT $1 OFFSET $2)t`, limit, offset)
}

func (s Service) SaveAchievement(ctx context.Context, user string, in AchievementInput) (string, error) {
	var err error
	if in.Title, err = clean(in.Title, 100, true); err != nil || in.Count < 0 || in.Count > 1000000000 {
		return "", apperror.ErrInvalid
	}
	if in.Icon, err = clean(in.Icon, 100, false); err != nil {
		return "", err
	}
	if in.Icon == "" {
		in.Icon = "TrophyIcon"
	}
	return s.insert(ctx, user, true, "PUBLIC_ACHIEVEMENT_CREATED", "public_achievement", func(tx pgx.Tx, id *string) error {
		return tx.QueryRow(ctx, `INSERT INTO public_achievements(title,count,icon,created_by) VALUES($1,$2,$3,$4) RETURNING id`, in.Title, in.Count, in.Icon, user).Scan(id)
	})
}

func (s Service) insert(ctx context.Context, user string, platformOnly bool, action, kind string, create func(pgx.Tx, *string) error) (string, error) {
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		if platformOnly {
			err = s.requirePlatform(ctx, tx, user)
		} else {
			err = s.requireContributor(ctx, tx, user)
		}
		if err != nil {
			return err
		}
		if err := create(tx, &id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, "", user, action, kind, id, nil)
	})
	return id, err
}

func (s Service) Delete(ctx context.Context, user, table, kind, action, id string) error {
	allowed := map[string]bool{"public_hero_slides": true, "public_events": true, "public_gallery_items": true, "public_achievements": true}
	if !allowed[table] {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		if table == "public_achievements" {
			err = s.requirePlatform(ctx, tx, user)
		} else {
			err = s.requireContributor(ctx, tx, user)
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE `+table+` SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, "", user, action, kind, id, nil)
	})
}

func (s Service) Reps(ctx context.Context, user string, featuredOnly bool, limit, offset int) (json.RawMessage, error) {
	if user != "" {
		if err := s.requirePlatform(ctx, s.Pool, user); err != nil {
			return nil, err
		}
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (
		SELECT a.id,u.display_name AS name,p.title AS role,b.name AS batch,''::text AS image_url,''::text AS linkedin,(f.assignment_id IS NOT NULL) AS "isFeaturedOnHome"
		FROM position_assignments a
		JOIN batch_positions p ON p.id=a.position_id AND p.batch_id=a.batch_id
		JOIN batches b ON b.id=a.batch_id
		JOIN users u ON u.id=a.user_id
		LEFT JOIN public_featured_representatives f ON f.assignment_id=a.id
		WHERE p.is_public AND b.status<>'ARCHIVED' AND u.status='ACTIVE' AND a.starts_at<=now() AND (a.ends_at IS NULL OR a.ends_at>now()) AND (NOT $1 OR f.assignment_id IS NOT NULL)
		ORDER BY p.sort_order,b.entry_year DESC,u.display_name,a.id LIMIT $2 OFFSET $3)t`, featuredOnly, limit, offset))
}

func (s Service) SetFeatured(ctx context.Context, user, assignment string, featured bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := s.requirePlatform(ctx, tx, user); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM position_assignments a JOIN batch_positions p ON p.id=a.position_id AND p.batch_id=a.batch_id JOIN batches b ON b.id=a.batch_id JOIN users u ON u.id=a.user_id WHERE a.id=$1 AND p.is_public AND b.status<>'ARCHIVED' AND u.status='ACTIVE' AND a.starts_at<=now() AND (a.ends_at IS NULL OR a.ends_at>now()))`, assignment).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return apperror.ErrNotFound
		}
		if featured {
			_, err := tx.Exec(ctx, `INSERT INTO public_featured_representatives(assignment_id,featured_by) VALUES($1,$2) ON CONFLICT(assignment_id) DO UPDATE SET featured_by=EXCLUDED.featured_by,featured_at=now()`, assignment, user)
			if err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(ctx, `DELETE FROM public_featured_representatives WHERE assignment_id=$1`, assignment)
			if err != nil {
				return err
			}
		}
		return audit.Record(ctx, tx, "", user, "PUBLIC_REP_FEATURE_CHANGED", "position_assignment", assignment, map[string]bool{"featured": featured})
	})
}

func (s Service) Socials(ctx context.Context, user string) (json.RawMessage, error) {
	return s.list(ctx, user, true, `SELECT COALESCE((SELECT row_to_json(t) FROM (SELECT whatsapp,facebook,instagram,youtube FROM public_social_links WHERE singleton=true)t),'{"whatsapp":"","facebook":"","instagram":"","youtube":""}'::json)`)
}

func (s Service) SaveSocials(ctx context.Context, user string, in SocialInput) error {
	var err error
	if in.WhatsApp, err = webURL(in.WhatsApp, false); err != nil {
		return err
	}
	if in.Facebook, err = webURL(in.Facebook, false); err != nil {
		return err
	}
	if in.Instagram, err = webURL(in.Instagram, false); err != nil {
		return err
	}
	if in.YouTube, err = webURL(in.YouTube, false); err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := s.requirePlatform(ctx, tx, user); err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO public_social_links(whatsapp,facebook,instagram,youtube,updated_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(singleton) DO UPDATE SET whatsapp=EXCLUDED.whatsapp,facebook=EXCLUDED.facebook,instagram=EXCLUDED.instagram,youtube=EXCLUDED.youtube,updated_by=EXCLUDED.updated_by,updated_at=now() RETURNING id`, in.WhatsApp, in.Facebook, in.Instagram, in.YouTube, user).Scan(&id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, "", user, "PUBLIC_SOCIAL_LINKS_UPDATED", "public_social_links", id, nil)
	})
}
