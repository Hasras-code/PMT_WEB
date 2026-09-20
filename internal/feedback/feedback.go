package feedback

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	Category  string `json:"category"`
	Message   string `json:"message"`
	Anonymous bool   `json:"is_anonymous"`
	Rating    *int   `json:"rating"`
}

func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if strings.TrimSpace(in.Message) == "" || len(in.Message) > 20000 || len(in.Category) > 100 || (in.Rating != nil && (*in.Rating < 1 || *in.Rating > 5)) {
		return "", apperror.ErrInvalid
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "feedback.create"); e != nil {
			return e
		}
		var author *string
		if !in.Anonymous {
			author = &user
		}
		return tx.QueryRow(ctx, `INSERT INTO feedback(batch_id,submitted_by,category,message,is_anonymous,rating) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, batch, author, in.Category, in.Message, in.Anonymous, in.Rating).Scan(&id)
	})
	return id, e
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "feedback.view"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,category,message,is_anonymous,rating,status,created_at FROM feedback WHERE batch_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3)t`, batch, limit, offset))
}
func (s Service) Status(ctx context.Context, user, batch, id, status string) error {
	if status != "REVIEWED" && status != "ARCHIVED" {
		return apperror.ErrInvalid
	}
	if e := authorization.Require(ctx, s.Pool, user, batch, "feedback.manage"); e != nil {
		return e
	}
	tag, e := s.Pool.Exec(ctx, `UPDATE feedback SET status=$3 WHERE batch_id=$1 AND id=$2 AND (status='NEW' OR status=$3 OR (status='REVIEWED' AND $3='ARCHIVED'))`, batch, id, status)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrConflict
	}
	return nil
}
