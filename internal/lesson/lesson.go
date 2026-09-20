package lesson

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"strings"
	"time"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	ModuleID        *string `json:"module_id"`
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	YoutubeVideoID  *string `json:"youtube_video_id"`
	LessonDate      *string `json:"lesson_date"`
	DurationSeconds *int    `json:"duration_seconds"`
}

func (in Input) Validate(create bool) error {
	if create && in.ModuleID == nil {
		return apperror.ErrInvalid
	}
	if in.ModuleID != nil {
		if len(*in.ModuleID) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.ModuleID) == "" {
			return apperror.ErrInvalid
		}
		if _, e := uuid.Parse(*in.ModuleID); e != nil {
			return apperror.ErrInvalid
		}
	}
	if create && in.Title == nil {
		return apperror.ErrInvalid
	}
	if in.Title != nil {
		if len(*in.Title) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.Title) == "" {
			return apperror.ErrInvalid
		}
	}
	if in.Description != nil {
		if len(*in.Description) > 20000 {
			return apperror.ErrInvalid
		}
	}
	if create && in.YoutubeVideoID == nil {
		return apperror.ErrInvalid
	}
	if in.YoutubeVideoID != nil {
		if len(*in.YoutubeVideoID) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.YoutubeVideoID) == "" {
			return apperror.ErrInvalid
		}

		if !regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`).MatchString(*in.YoutubeVideoID) {
			return apperror.ErrInvalid
		}
	}
	if in.LessonDate != nil {
		if len(*in.LessonDate) > 20000 {
			return apperror.ErrInvalid
		}
		if _, e := time.Parse("2006-01-02", *in.LessonDate); e != nil {
			return apperror.ErrInvalid
		}
	}
	if in.DurationSeconds != nil && (*in.DurationSeconds < 1 || *in.DurationSeconds > 100000) {
		return apperror.ErrInvalid
	}
	return nil
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "lesson.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "lesson.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,batch_id,module_id,title,description,youtube_video_id,lesson_date,duration_seconds,status,published_at,created_at,updated_at FROM recorded_lessons WHERE batch_id=$1  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $4)t`, batch, limit, manage, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "lesson.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "lesson.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT id,batch_id,module_id,title,description,youtube_video_id,lesson_date,duration_seconds,status,published_at,created_at,updated_at FROM recorded_lessons WHERE batch_id=$1 AND id=$2  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')))t`, batch, id, manage))
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "lesson.manage"); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `INSERT INTO recorded_lessons(batch_id,created_by,module_id,title,description,youtube_video_id,lesson_date,duration_seconds) VALUES($1,$2,COALESCE($3::uuid,NULL),COALESCE($4,''),COALESCE($5,''),COALESCE($6,''),$7::text::date,$8::integer) RETURNING id`, batch, user, in.ModuleID, in.Title, in.Description, in.YoutubeVideoID, in.LessonDate, in.DurationSeconds).Scan(&id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "LESSON_CREATED", "lesson", id, nil)
	})
	return id, e
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.Validate(false); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "lesson.manage"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE recorded_lessons SET module_id=COALESCE($3,module_id),title=COALESCE($4,title),description=COALESCE($5,description),youtube_video_id=COALESCE($6,youtube_video_id),lesson_date=COALESCE($7::text::date,lesson_date),duration_seconds=COALESCE($8,duration_seconds),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, in.ModuleID, in.Title, in.Description, in.YoutubeVideoID, in.LessonDate, in.DurationSeconds)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "LESSON_UPDATED", "lesson", id, nil)
	})
}
func (s Service) Transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		permission := "lesson.manage"
		target := "ARCHIVED"
		if publish {
			permission = "lesson.manage"
			target = "PUBLISHED"
		}
		if e := authorization.Write(ctx, tx, user, batch, permission); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM recorded_lessons WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); e != nil {
			return e
		}
		if old == target {
			return nil
		}
		if publish && old != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE recorded_lessons SET status=$3,updated_at=now(),published_at=CASE WHEN $3='PUBLISHED' THEN now() ELSE published_at END WHERE batch_id=$1 AND id=$2`, batch, id, target); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "LESSON_"+target, "lesson", id, nil)
	})
}
