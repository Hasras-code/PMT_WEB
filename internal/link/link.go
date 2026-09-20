package link

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
	"net/url"
	"strings"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	ModuleID    *string `json:"module_id"`
	Title       *string `json:"title"`
	URL         *string `json:"url"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
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
	if create && in.URL == nil {
		return apperror.ErrInvalid
	}
	if in.URL != nil {
		if len(*in.URL) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.URL) == "" {
			return apperror.ErrInvalid
		}
		u, e := url.Parse(*in.URL)
		if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
			return apperror.ErrInvalid
		}
	}
	if in.Description != nil {
		if len(*in.Description) > 20000 {
			return apperror.ErrInvalid
		}
	}
	if in.Category != nil {
		if len(*in.Category) > 20000 {
			return apperror.ErrInvalid
		}
	}
	return nil
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "link.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "link.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,batch_id,module_id,title,url,description,category,status,created_at,updated_at FROM useful_links WHERE batch_id=$1  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $4)t`, batch, limit, manage, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "link.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "link.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT id,batch_id,module_id,title,url,description,category,status,created_at,updated_at FROM useful_links WHERE batch_id=$1 AND id=$2  AND (status='PUBLISHED' OR ($3 AND status='DRAFT')))t`, batch, id, manage))
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "link.manage"); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `INSERT INTO useful_links(batch_id,created_by,module_id,title,url,description,category) VALUES($1,$2,COALESCE($3::uuid,NULL),COALESCE($4,''),COALESCE($5,''),COALESCE($6,''),COALESCE($7,'')) RETURNING id`, batch, user, in.ModuleID, in.Title, in.URL, in.Description, in.Category).Scan(&id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "LINK_CREATED", "link", id, nil)
	})
	return id, e
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.Validate(false); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "link.manage"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE useful_links SET module_id=COALESCE($3,module_id),title=COALESCE($4,title),url=COALESCE($5,url),description=COALESCE($6,description),category=COALESCE($7,category),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, in.ModuleID, in.Title, in.URL, in.Description, in.Category)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "LINK_UPDATED", "link", id, nil)
	})
}
func (s Service) Transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		permission := "link.manage"
		target := "ARCHIVED"
		if publish {
			permission = "link.manage"
			target = "PUBLISHED"
		}
		if e := authorization.Write(ctx, tx, user, batch, permission); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM useful_links WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); e != nil {
			return e
		}
		if old == target {
			return nil
		}
		if publish && old != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE useful_links SET status=$3,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, target); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "LINK_"+target, "link", id, nil)
	})
}
