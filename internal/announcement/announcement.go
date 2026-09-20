package announcement

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
	"strings"
	"time"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	ModuleID  *string    `json:"module_id"`
	Title     *string    `json:"title"`
	Body      *string    `json:"body"`
	Priority  *string    `json:"priority"`
	Pinned    *bool      `json:"pinned"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (in Input) Validate(create bool) error {
	if in.ModuleID != nil {
		if len(*in.ModuleID) > 20000 {
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
	if create && in.Body == nil {
		return apperror.ErrInvalid
	}
	if in.Body != nil {
		if len(*in.Body) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.Body) == "" {
			return apperror.ErrInvalid
		}
	}
	if in.Priority != nil {
		if len(*in.Priority) > 20000 {
			return apperror.ErrInvalid
		}
		if *in.Priority != "NORMAL" && *in.Priority != "IMPORTANT" && *in.Priority != "URGENT" {
			return apperror.ErrInvalid
		}
	}
	return nil
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "announcement.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "announcement.update")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,batch_id,module_id,title,body,priority,pinned,expires_at,status,published_at,created_at,updated_at FROM announcements WHERE batch_id=$1  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')) AND (expires_at IS NULL OR expires_at>now() OR $3) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $4)t`, batch, limit, manage, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "announcement.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "announcement.update")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT id,batch_id,module_id,title,body,priority,pinned,expires_at,status,published_at,created_at,updated_at FROM announcements WHERE batch_id=$1 AND id=$2  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')) AND (expires_at IS NULL OR expires_at>now() OR $3))t`, batch, id, manage))
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "announcement.create"); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `INSERT INTO announcements(batch_id,created_by,module_id,title,body,priority,pinned,expires_at) VALUES($1,$2,COALESCE($3::uuid,NULL),COALESCE($4,''),COALESCE($5,''),COALESCE($6,'NORMAL'),COALESCE($7,false),$8::timestamptz) RETURNING id`, batch, user, in.ModuleID, in.Title, in.Body, in.Priority, in.Pinned, in.ExpiresAt).Scan(&id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "ANNOUNCEMENT_CREATED", "announcement", id, nil)
	})
	return id, e
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.Validate(false); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "announcement.update"); e != nil {
			return e
		}
		var state string
		if e := tx.QueryRow(ctx, `SELECT status FROM announcements WHERE batch_id=$1 AND id=$2`, batch, id).Scan(&state); e != nil {
			return e
		}
		if state == "PUBLISHED" {
			if e := authorization.Require(ctx, tx, user, batch, "announcement.publish"); e != nil {
				return e
			}
		}
		tag, e := tx.Exec(ctx, `UPDATE announcements SET module_id=COALESCE($3,module_id),title=COALESCE($4,title),body=COALESCE($5,body),priority=COALESCE($6,priority),pinned=COALESCE($7,pinned),expires_at=COALESCE($8,expires_at),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, in.ModuleID, in.Title, in.Body, in.Priority, in.Pinned, in.ExpiresAt)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "ANNOUNCEMENT_UPDATED", "announcement", id, nil)
	})
}
func (s Service) Transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		permission := "announcement.delete"
		target := "ARCHIVED"
		if publish {
			permission = "announcement.publish"
			target = "PUBLISHED"
		}
		if e := authorization.Write(ctx, tx, user, batch, permission); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM announcements WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); e != nil {
			return e
		}
		if old == target {
			return nil
		}
		if publish && old != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE announcements SET status=$3,updated_at=now(),published_at=CASE WHEN $3='PUBLISHED' THEN now() ELSE published_at END WHERE batch_id=$1 AND id=$2`, batch, id, target); e != nil {
			return e
		}
		if publish {
			if _, e := tx.Exec(ctx, `INSERT INTO notification_events(batch_id,entity_id,event_type,title) SELECT batch_id,id,'ANNOUNCEMENT_PUBLISHED',title FROM announcements WHERE id=$1 ON CONFLICT DO NOTHING`, id); e != nil {
				return e
			}
		}
		return audit.Record(ctx, tx, batch, user, "ANNOUNCEMENT_"+target, "announcement", id, nil)
	})
}
