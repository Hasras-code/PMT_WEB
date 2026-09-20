package complaint

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
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	Category  string `json:"category"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Anonymous bool   `json:"is_anonymous"`
}

func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if strings.TrimSpace(in.Subject) == "" || strings.TrimSpace(in.Message) == "" || in.Category == "" || len(in.Subject) > 300 || len(in.Message) > 20000 || len(in.Category) > 100 {
		return "", apperror.ErrInvalid
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "complaint.create"); e != nil {
			return e
		}
		var author *string
		if !in.Anonymous {
			author = &user
		}
		return tx.QueryRow(ctx, `INSERT INTO complaints(batch_id,submitted_by,category,subject,message,is_anonymous) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, batch, author, in.Category, in.Subject, in.Message, in.Anonymous).Scan(&id)
	})
	return id, e
}
func (s Service) access(ctx context.Context, q authorization.Querier, user, batch, id string) (bool, error) {
	if e := authorization.Require(ctx, q, user, batch, "complaint.view_own"); e != nil {
		return false, e
	}
	manager, e := authorization.Can(ctx, q, user, batch, "complaint.view_all")
	if e != nil {
		return false, e
	}
	var owner *string
	if e = q.QueryRow(ctx, `SELECT submitted_by FROM complaints WHERE batch_id=$1 AND id=$2`, batch, id).Scan(&owner); e != nil {
		return false, db.Error(e)
	}
	if !manager && (owner == nil || *owner != user) {
		return false, apperror.ErrNotFound
	}
	return manager, nil
}
func (s Service) List(ctx context.Context, user, batch string, mine bool, limit, offset int) (json.RawMessage, error) {
	permission := "complaint.view_all"
	if mine {
		permission = "complaint.view_own"
	}
	if e := authorization.Require(ctx, s.Pool, user, batch, permission); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,category,subject,is_anonymous,status,assigned_to,resolved_at,created_at FROM complaints WHERE batch_id=$1 AND (NOT $2 OR submitted_by=$3::uuid) ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5)t`, batch, mine, user, limit, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if _, e := s.access(ctx, s.Pool, user, batch, id); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM(SELECT id,batch_id,submitted_by,category,subject,message,is_anonymous,status,assigned_to,resolved_at,created_at,updated_at FROM complaints WHERE batch_id=$1 AND id=$2)t`, batch, id))
}
func (s Service) Messages(ctx context.Context, user, batch, id string, limit, offset int) (json.RawMessage, error) {
	if _, e := s.access(ctx, s.Pool, user, batch, id); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,author_user_id,author_type,message,created_at FROM complaint_messages WHERE batch_id=$1 AND complaint_id=$2 ORDER BY created_at,id LIMIT $3 OFFSET $4)t`, batch, id, limit, offset))
}
func (s Service) Reply(ctx context.Context, user, batch, id, message string) error {
	if strings.TrimSpace(message) == "" || len(message) > 20000 {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "complaint.view_own"); e != nil {
			return e
		}
		manager, e := s.access(ctx, tx, user, batch, id)
		if e != nil {
			return e
		}
		kind := "STUDENT"
		if manager {
			if e := authorization.Require(ctx, tx, user, batch, "complaint.respond"); e != nil {
				return e
			}
			kind = "ADMIN"
		}
		var status string
		if e = tx.QueryRow(ctx, `SELECT status FROM complaints WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&status); e != nil {
			return e
		}
		if status == "CLOSED" || status == "RESOLVED" {
			return apperror.ErrConflict
		}
		_, e = tx.Exec(ctx, `INSERT INTO complaint_messages(batch_id,complaint_id,author_user_id,author_type,message) VALUES($1,$2,$3,$4,$5)`, batch, id, user, kind, message)
		return e
	})
}
func ValidTransition(old, next string) bool {
	return (old == "OPEN" && next == "IN_REVIEW") || (old == "IN_REVIEW" && next == "RESOLVED") || (old == "RESOLVED" && (next == "CLOSED" || next == "IN_REVIEW"))
}
func (s Service) Transition(ctx context.Context, user, batch, id, next string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		perm := "complaint.respond"
		if next == "RESOLVED" || next == "CLOSED" {
			perm = "complaint.resolve"
		}
		if e := authorization.Write(ctx, tx, user, batch, perm); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM complaints WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); e != nil {
			return e
		}
		if old == next {
			return nil
		}
		if !ValidTransition(old, next) {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE complaints SET status=$3,resolved_at=CASE WHEN $3='RESOLVED' THEN now() WHEN $3='IN_REVIEW' THEN NULL ELSE resolved_at END,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, next); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "COMPLAINT_"+next, "complaint", id, map[string]string{"previous": old})
	})
}
func (s Service) Assign(ctx context.Context, user, batch, id, assignee string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "complaint.respond"); e != nil {
			return e
		}
		if e := authorization.Require(ctx, tx, assignee, batch, "complaint.respond"); e != nil {
			return apperror.ErrInvalid
		}
		tag, e := tx.Exec(ctx, `UPDATE complaints SET assigned_to=$3,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, assignee)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "COMPLAINT_ASSIGNED", "complaint", id, nil)
	})
}
