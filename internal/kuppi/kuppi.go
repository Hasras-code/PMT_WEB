package kuppi

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ Pool *pgxpool.Pool }

type Input struct {
	ModuleID        *string `json:"module_id"`
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	YouTubeURL      *string `json:"youtube_url"`
	RecordedAt      *string `json:"recorded_at"`
	DurationSeconds *int    `json:"duration_seconds"`
	SortOrder       *int    `json:"sort_order"`
}

type Filter struct {
	ModuleID        string
	Search          string
	DateFrom        string
	DateTo          string
	IncludeArchived bool
}

const projection = `json_build_object(
	'id', k.id,
	'batch_id', k.batch_id,
	'module', json_build_object('id', m.id, 'code', m.module_code, 'name', m.name),
	'title', k.title,
	'description', k.description,
	'youtube_video_id', k.youtube_video_id,
	'embed_url', 'https://www.youtube-nocookie.com/embed/' || k.youtube_video_id,
	'watch_url', 'https://www.youtube.com/watch?v=' || k.youtube_video_id,
	'recorded_at', k.recorded_at,
	'duration_seconds', k.duration_seconds,
	'sort_order', k.sort_order,
	'status', k.status,
	'published_at', k.published_at,
	'created_at', k.created_at,
	'updated_at', k.updated_at
)`

func coded(base error, code string) error { return apperror.WithCode(base, code) }

func (in Input) validate(create bool) error {
	if create && (in.ModuleID == nil || in.Title == nil || in.YouTubeURL == nil) {
		return apperror.ErrInvalid
	}
	if in.ModuleID != nil {
		if _, err := uuid.Parse(strings.TrimSpace(*in.ModuleID)); err != nil {
			return coded(apperror.ErrInvalid, "module_not_found")
		}
	}
	if in.Title != nil && (strings.TrimSpace(*in.Title) == "" || len(*in.Title) > 200) {
		return apperror.ErrInvalid
	}
	if in.Description != nil && len(*in.Description) > 20000 {
		return apperror.ErrInvalid
	}
	if in.YouTubeURL != nil {
		if _, err := ParseYouTubeURL(*in.YouTubeURL); err != nil {
			return err
		}
	}
	if in.RecordedAt != nil {
		if _, err := time.Parse(time.RFC3339, *in.RecordedAt); err != nil {
			return apperror.ErrInvalid
		}
	}
	if in.DurationSeconds != nil && (*in.DurationSeconds < 1 || *in.DurationSeconds > 86400) {
		return apperror.ErrInvalid
	}
	if in.SortOrder != nil && *in.SortOrder < 0 {
		return apperror.ErrInvalid
	}
	return nil
}

func (f *Filter) validate() error {
	if f.ModuleID != "" {
		if _, err := uuid.Parse(f.ModuleID); err != nil {
			return apperror.ErrInvalid
		}
	}
	f.Search = strings.TrimSpace(f.Search)
	if len(f.Search) > 200 {
		return apperror.ErrInvalid
	}
	for _, value := range []string{f.DateFrom, f.DateTo} {
		if value != "" {
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				return apperror.ErrInvalid
			}
		}
	}
	return nil
}

func (s Service) canView(ctx context.Context, q authorization.Querier, user, batch string) (bool, error) {
	return authorization.CanWithPlatform(ctx, q, user, batch, "kuppi.view", "platform_user.manage")
}

func (s Service) canManage(ctx context.Context, q authorization.Querier, user, batch string) (bool, error) {
	return authorization.CanWithPlatform(ctx, q, user, batch, "kuppi.update", "platform_user.manage")
}

func (s Service) List(ctx context.Context, user, batch string, filter Filter, limit, offset int) (json.RawMessage, error) {
	if err := filter.validate(); err != nil {
		return nil, err
	}
	if ok, err := s.canView(ctx, s.Pool, user, batch); err != nil || !ok {
		if err != nil {
			return nil, err
		}
		return nil, apperror.ErrForbidden
	}
	manage, err := s.canManage(ctx, s.Pool, user, batch)
	if err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(value),'[]') FROM (
		SELECT `+projection+` AS value FROM kuppi_recordings k JOIN modules m ON m.id=k.module_id AND m.batch_id=k.batch_id
		WHERE k.batch_id=$1
		AND ($2 OR k.status <> 'ARCHIVED')
		AND (k.status='PUBLISHED' OR ($3 AND k.status='DRAFT') OR ($3 AND $2 AND k.status='ARCHIVED'))
		AND ($4='' OR k.module_id=NULLIF($4,'')::uuid)
		AND ($5='' OR k.title ILIKE '%'||$5||'%' OR k.description ILIKE '%'||$5||'%' OR m.module_code ILIKE '%'||$5||'%' OR m.name ILIKE '%'||$5||'%')
		AND ($6='' OR k.recorded_at >= $6::timestamptz)
		AND ($7='' OR k.recorded_at <= $7::timestamptz)
		ORDER BY COALESCE(k.recorded_at,k.published_at,k.created_at) DESC,k.id DESC LIMIT $8 OFFSET $9
	)t`, batch, filter.IncludeArchived, manage, filter.ModuleID, filter.Search, filter.DateFrom, filter.DateTo, limit, offset))
}

func (s Service) Get(ctx context.Context, user, batch, id string, includeArchived bool) (json.RawMessage, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, apperror.ErrInvalid
	}
	if ok, err := s.canView(ctx, s.Pool, user, batch); err != nil || !ok {
		if err != nil {
			return nil, err
		}
		return nil, apperror.ErrForbidden
	}
	manage, err := s.canManage(ctx, s.Pool, user, batch)
	if err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT `+projection+` FROM kuppi_recordings k JOIN modules m ON m.id=k.module_id AND m.batch_id=k.batch_id WHERE k.batch_id=$1 AND k.id=$2 AND (k.status='PUBLISHED' OR ($3 AND k.status='DRAFT') OR ($3 AND $4 AND k.status='ARCHIVED'))`, batch, id, manage, includeArchived))
}

func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if err := in.validate(true); err != nil {
		return "", err
	}
	id := uuid.NewString()
	videoID, _ := ParseYouTubeURL(*in.YouTubeURL)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.RequireWithPlatform(ctx, tx, user, batch, "kuppi.create", "platform_user.manage"); err != nil {
			return err
		}
		var active bool
		if err := tx.QueryRow(ctx, `SELECT status='ACTIVE' FROM modules WHERE batch_id=$1 AND id=$2`, batch, strings.TrimSpace(*in.ModuleID)).Scan(&active); err != nil {
			return coded(apperror.ErrNotFound, "module_not_found")
		}
		if !active {
			return coded(apperror.ErrConflict, "module_not_found")
		}
		var duplicate bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM kuppi_recordings WHERE batch_id=$1 AND module_id=$2 AND youtube_video_id=$3 AND status<>'ARCHIVED')`, batch, *in.ModuleID, videoID).Scan(&duplicate); err != nil {
			return err
		}
		if duplicate {
			return coded(apperror.ErrConflict, "duplicate_kuppi")
		}
		_, err := tx.Exec(ctx, `INSERT INTO kuppi_recordings(id,batch_id,module_id,created_by,title,description,youtube_video_id,recorded_at,duration_seconds,sort_order) VALUES($1,$2,$3,$4,$5,COALESCE($6,''),$7,$8::timestamptz,$9,COALESCE($10,0))`, id, batch, *in.ModuleID, user, strings.TrimSpace(*in.Title), in.Description, videoID, in.RecordedAt, in.DurationSeconds, in.SortOrder)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "KUPPI_CREATED", "kuppi", id, nil)
	})
	return id, err
}

func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if err := in.validate(false); err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.RequireWithPlatform(ctx, tx, user, batch, "kuppi.update", "platform_user.manage"); err != nil {
			return err
		}
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM kuppi_recordings WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&status); err != nil {
			return err
		}
		if status == "ARCHIVED" {
			return coded(apperror.ErrConflict, "kuppi_already_archived")
		}
		var currentModuleID, currentVideoID string
		if err := tx.QueryRow(ctx, `SELECT module_id,youtube_video_id FROM kuppi_recordings WHERE batch_id=$1 AND id=$2`, batch, id).Scan(&currentModuleID, &currentVideoID); err != nil {
			return err
		}
		moduleID := (*string)(nil)
		if in.ModuleID != nil {
			moduleID = in.ModuleID
		}
		if moduleID != nil {
			var active bool
			if err := tx.QueryRow(ctx, `SELECT status='ACTIVE' FROM modules WHERE batch_id=$1 AND id=$2`, batch, *moduleID).Scan(&active); err != nil {
				return coded(apperror.ErrNotFound, "module_not_found")
			}
			if !active {
				return coded(apperror.ErrConflict, "module_not_found")
			}
		}
		var videoID *string
		if in.YouTubeURL != nil {
			parsed, _ := ParseYouTubeURL(*in.YouTubeURL)
			videoID = &parsed
		}
		effectiveModuleID := currentModuleID
		if moduleID != nil {
			effectiveModuleID = *moduleID
		}
		effectiveVideoID := currentVideoID
		if videoID != nil {
			effectiveVideoID = *videoID
		}
		var duplicate bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM kuppi_recordings WHERE batch_id=$1 AND module_id=$2 AND youtube_video_id=$3 AND id<>$4 AND status<>'ARCHIVED')`, batch, effectiveModuleID, effectiveVideoID, id).Scan(&duplicate); err != nil {
			return err
		}
		if duplicate {
			return coded(apperror.ErrConflict, "duplicate_kuppi")
		}
		_, err := tx.Exec(ctx, `UPDATE kuppi_recordings SET module_id=COALESCE($3,module_id),title=COALESCE($4,title),description=COALESCE($5,description),youtube_video_id=COALESCE($6,youtube_video_id),recorded_at=COALESCE($7::timestamptz,recorded_at),duration_seconds=COALESCE($8,duration_seconds),sort_order=COALESCE($9,sort_order),updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, moduleID, in.Title, in.Description, videoID, in.RecordedAt, in.DurationSeconds, in.SortOrder)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "KUPPI_UPDATED", "kuppi", id, nil)
	})
}

func (s Service) Publish(ctx context.Context, user, batch, id string) error {
	return s.transition(ctx, user, batch, id, true)
}

func (s Service) Archive(ctx context.Context, user, batch, id string) error {
	return s.transition(ctx, user, batch, id, false)
}

func (s Service) transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		permission := "kuppi.archive"
		if publish {
			permission = "kuppi.publish"
		}
		if err := authorization.RequireWithPlatform(ctx, tx, user, batch, permission, "platform_user.manage"); err != nil {
			return err
		}
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM kuppi_recordings WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&status); err != nil {
			return err
		}
		if publish {
			if status == "PUBLISHED" {
				return nil
			}
			if status == "ARCHIVED" {
				return coded(apperror.ErrConflict, "kuppi_already_archived")
			}
			if status != "DRAFT" {
				return coded(apperror.ErrConflict, "invalid_kuppi_transition")
			}
			if _, err := tx.Exec(ctx, `UPDATE kuppi_recordings SET status='PUBLISHED',published_at=now(),updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO notification_events(batch_id,entity_id,event_type,title) VALUES($1,$2,'KUPPI_PUBLISHED','New Kuppi recording') ON CONFLICT DO NOTHING`, batch, id); err != nil {
				return err
			}
			return audit.Record(ctx, tx, batch, user, "KUPPI_PUBLISHED", "kuppi", id, nil)
		}
		if status == "ARCHIVED" {
			return nil
		}
		if status != "DRAFT" && status != "PUBLISHED" {
			return coded(apperror.ErrConflict, "invalid_kuppi_transition")
		}
		if _, err := tx.Exec(ctx, `UPDATE kuppi_recordings SET status='ARCHIVED',archived_at=now(),updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "KUPPI_ARCHIVED", "kuppi", id, nil)
	})
}
