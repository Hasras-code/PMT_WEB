package position

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	Public      *bool   `json:"is_public"`
}

func (s Service) List(ctx context.Context, user, batch, id string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "position.view"); e != nil {
		return nil, e
	}
	b, e := db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,title,description,sort_order,is_public FROM batch_positions WHERE batch_id=$1 AND ($2='' OR id=NULLIF($2,'')::uuid) ORDER BY sort_order,id LIMIT $3 OFFSET $4)t`, batch, id, limit, offset))
	if id != "" {
		return db.One(b, e)
	}
	return b, e
}
func (s Service) Save(ctx context.Context, user, batch, id string, in Input) (string, error) {
	if (id == "" && in.Title == nil) || (in.Title != nil && (*in.Title == "" || len(*in.Title) > 200)) || (in.Description != nil && len(*in.Description) > 20000) {
		return "", apperror.ErrInvalid
	}
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "position.manage"); e != nil {
			return e
		}
		if id == "" {
			return tx.QueryRow(ctx, `INSERT INTO batch_positions(batch_id,title,description,sort_order,is_public) VALUES($1,$2,COALESCE($3,''),COALESCE($4,0),COALESCE($5,true)) RETURNING id`, batch, in.Title, in.Description, in.SortOrder, in.Public).Scan(&id)
		}
		tag, e := tx.Exec(ctx, `UPDATE batch_positions SET title=COALESCE($3,title),description=COALESCE($4,description),sort_order=COALESCE($5,sort_order),is_public=COALESCE($6,is_public) WHERE batch_id=$1 AND id=$2`, batch, id, in.Title, in.Description, in.SortOrder, in.Public)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return nil
	})
	return id, e
}
func (s Service) Hide(ctx context.Context, user, batch, id string) error {
	if e := authorization.Require(ctx, s.Pool, user, batch, "position.manage"); e != nil {
		return e
	}
	tag, e := s.Pool.Exec(ctx, `UPDATE batch_positions SET is_public=false WHERE batch_id=$1 AND id=$2`, batch, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
func (s Service) Assign(ctx context.Context, user, batch, id, target string, start time.Time, end *time.Time) (string, error) {
	if start.IsZero() || (end != nil && end.Before(start)) {
		return "", apperror.ErrInvalid
	}
	var assignment string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "position.manage"); e != nil {
			return e
		}
		var active bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM batch_memberships WHERE batch_id=$1 AND user_id=$2 AND status='ACTIVE')`, batch, target).Scan(&active); e != nil {
			return e
		}
		if !active {
			return apperror.ErrInvalid
		}
		return tx.QueryRow(ctx, `INSERT INTO position_assignments(batch_id,position_id,user_id,starts_at,ends_at) VALUES($1,$2,$3,$4,$5) RETURNING id`, batch, id, target, start, end).Scan(&assignment)
	})
	return assignment, e
}
func (s Service) End(ctx context.Context, user, batch, id, assignment string) error {
	if e := authorization.Require(ctx, s.Pool, user, batch, "position.manage"); e != nil {
		return e
	}
	tag, e := s.Pool.Exec(ctx, `UPDATE position_assignments SET ends_at=GREATEST(now(),starts_at) WHERE batch_id=$1 AND position_id=$2 AND id=$3 AND (ends_at IS NULL OR ends_at>now())`, batch, id, assignment)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
func (s Service) Assignments(ctx context.Context, user, batch, id string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "position.view"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT a.id,a.user_id,u.display_name,a.starts_at,a.ends_at FROM position_assignments a JOIN users u ON u.id=a.user_id WHERE a.batch_id=$1 AND a.position_id=$2 ORDER BY a.starts_at DESC,a.id DESC LIMIT $3 OFFSET $4)t`, batch, id, limit, offset))
}
func (s Service) Public(ctx context.Context, slug string, limit, offset int) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT p.id,p.title,p.description,p.sort_order,COALESCE((SELECT json_agg(json_build_object('display_name',u.display_name,'starts_at',a.starts_at,'ends_at',a.ends_at)) FROM position_assignments a JOIN users u ON u.id=a.user_id WHERE a.position_id=p.id AND a.batch_id=p.batch_id AND a.starts_at<=now() AND (a.ends_at IS NULL OR a.ends_at>now()) AND u.status='ACTIVE'),'[]') AS assignments FROM batch_positions p JOIN batches b ON b.id=p.batch_id WHERE b.slug=$1 AND b.status<>'ARCHIVED' AND p.is_public ORDER BY p.sort_order,p.id LIMIT $2 OFFSET $3)t`, slug, limit, offset))
}
