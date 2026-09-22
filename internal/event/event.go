package event

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Location    *string    `json:"location"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	Visibility  *string    `json:"visibility"`
}

func (in Input) Validate(create bool) error {
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
	if in.Location != nil {
		if len(*in.Location) > 20000 {
			return apperror.ErrInvalid
		}
	}
	if create && in.StartsAt == nil {
		return apperror.ErrInvalid
	}
	if in.Visibility != nil {
		if len(*in.Visibility) > 20000 {
			return apperror.ErrInvalid
		}
		if *in.Visibility != "PUBLIC" && *in.Visibility != "MEMBERS_ONLY" {
			return apperror.ErrInvalid
		}
	}
	return nil
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "event.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "event.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,batch_id,title,description,location,starts_at,ends_at,visibility,status,published_at,created_at,updated_at FROM events WHERE batch_id=$1  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $4)t`, batch, limit, manage, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "event.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "event.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT id,batch_id,title,description,location,starts_at,ends_at,visibility,status,published_at,created_at,updated_at FROM events WHERE batch_id=$1 AND id=$2  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')))t`, batch, id, manage))
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var e error
		id, e = CreateTx(ctx, tx, user, batch, in)
		return e
	})
	return id, e
}

// CreateTx creates an event inside an existing transaction. It is used by
// workflows such as event-fund creation that must either complete together or
// leave no partial event behind.
func CreateTx(ctx context.Context, tx pgx.Tx, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	if e := authorization.Write(ctx, tx, user, batch, "event.manage"); e != nil {
		return "", e
	}
	var id string
	if e := tx.QueryRow(ctx, `INSERT INTO events(batch_id,created_by,title,description,location,starts_at,ends_at,visibility) VALUES($1,$2,COALESCE($3,''),COALESCE($4,''),COALESCE($5,''),$6::timestamptz,$7::timestamptz,COALESCE($8,'MEMBERS_ONLY')) RETURNING id`, batch, user, in.Title, in.Description, in.Location, in.StartsAt, in.EndsAt, in.Visibility).Scan(&id); e != nil {
		return "", e
	}
	if e := audit.Record(ctx, tx, batch, user, "EVENT_CREATED", "event", id, nil); e != nil {
		return "", e
	}
	return id, nil
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.Validate(false); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "event.manage"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE events SET title=COALESCE($3,title),description=COALESCE($4,description),location=COALESCE($5,location),starts_at=COALESCE($6,starts_at),ends_at=COALESCE($7,ends_at),visibility=COALESCE($8,visibility),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, in.Title, in.Description, in.Location, in.StartsAt, in.EndsAt, in.Visibility)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "EVENT_UPDATED", "event", id, nil)
	})
}
func (s Service) Transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		permission := "event.manage"
		target := "ARCHIVED"
		if publish {
			permission = "event.manage"
			target = "PUBLISHED"
		}
		if e := authorization.Write(ctx, tx, user, batch, permission); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM events WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); e != nil {
			return e
		}
		if old == target {
			return nil
		}
		if publish && old != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE events SET status=$3,updated_at=now(),published_at=CASE WHEN $3='PUBLISHED' THEN now() ELSE published_at END WHERE batch_id=$1 AND id=$2`, batch, id, target); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "EVENT_"+target, "event", id, nil)
	})
}
