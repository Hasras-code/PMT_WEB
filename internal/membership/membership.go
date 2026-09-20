package membership

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ Pool *pgxpool.Pool }

func (s Service) Create(ctx context.Context, actor, batch, user string, operator bool) (string, error) {
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if !operator {
			if e := authorization.Write(ctx, tx, actor, batch, "membership.manage"); e != nil {
				return e
			}
		}
		var active bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND status='ACTIVE')`, user).Scan(&active); e != nil {
			return e
		}
		if !active {
			return apperror.ErrConflict
		}
		if e := tx.QueryRow(ctx, `INSERT INTO batch_memberships(batch_id,user_id) VALUES($1,$2) RETURNING id`, batch, user).Scan(&id); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `INSERT INTO membership_roles(membership_id,role_id,assigned_by) SELECT $1,id,NULLIF($2,'')::uuid FROM roles WHERE code='STUDENT'`, id, actor); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, actor, "MEMBERSHIP_CREATED", "membership", id, map[string]string{"role": "STUDENT"})
	})
	return id, e
}
func (s Service) Role(ctx context.Context, actor, batch, member, role string, remove, operator bool) error {
	if role == "STUDENT" || role == "PLATFORM_ADMIN" {
		return apperror.ErrConflict
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if operator {
			var id string
			if e := tx.QueryRow(ctx, `SELECT id FROM batches WHERE id=$1 FOR UPDATE`, batch).Scan(&id); e != nil {
				return e
			}
		} else if e := authorization.Write(ctx, tx, actor, batch, "role.assign"); e != nil {
			return e
		}
		var status string
		if e := tx.QueryRow(ctx, `SELECT m.status FROM batch_memberships m JOIN users u ON u.id=m.user_id WHERE m.id=$1 AND m.batch_id=$2 AND u.status='ACTIVE'`, member, batch).Scan(&status); e != nil {
			return e
		}
		if status != "ACTIVE" {
			return apperror.ErrConflict
		}
		var rid string
		if e := tx.QueryRow(ctx, `SELECT id FROM roles WHERE code=$1 AND scope='BATCH'`, role).Scan(&rid); e != nil {
			return apperror.ErrInvalid
		}
		if remove && role == "BATCH_REP" && !operator {
			if e := protectRep(ctx, tx, batch, member); e != nil {
				return e
			}
		}
		action := "ROLE_ASSIGNED"
		var changed int64
		if remove {
			tag, e := tx.Exec(ctx, `DELETE FROM membership_roles WHERE membership_id=$1 AND role_id=$2`, member, rid)
			if e != nil {
				return e
			}
			changed = tag.RowsAffected()
			action = "ROLE_REMOVED"
		} else {
			tag, e := tx.Exec(ctx, `INSERT INTO membership_roles(membership_id,role_id,assigned_by) VALUES($1,$2,NULLIF($3,'')::uuid) ON CONFLICT DO NOTHING`, member, rid, actor)
			if e != nil {
				return e
			}
			changed = tag.RowsAffected()
		}
		if changed == 0 {
			return nil
		}
		return audit.Record(ctx, tx, batch, actor, action, "membership", member, map[string]any{"role": role, "operator": operator})
	})
}
func protectRep(ctx context.Context, tx pgx.Tx, batch, member string) error {
	var target bool
	var n int
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM membership_roles mr JOIN roles r ON r.id=mr.role_id WHERE mr.membership_id=$1 AND r.code='BATCH_REP')`, member).Scan(&target); e != nil {
		return e
	}
	if !target {
		return nil
	}
	if e := tx.QueryRow(ctx, `SELECT count(*) FROM batch_memberships m JOIN membership_roles mr ON mr.membership_id=m.id JOIN roles r ON r.id=mr.role_id JOIN users u ON u.id=m.user_id WHERE m.batch_id=$1 AND m.status='ACTIVE' AND u.status='ACTIVE' AND r.code='BATCH_REP'`, batch).Scan(&n); e != nil {
		return e
	}
	if n <= 1 {
		return apperror.ErrConflict
	}
	return nil
}
func (s Service) SetStatus(ctx context.Context, actor, batch, member, status string) error {
	switch status {
	case "ACTIVE", "SUSPENDED", "LEFT", "GRADUATED":
	default:
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, actor, batch, "membership.manage"); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM batch_memberships WHERE id=$1 AND batch_id=$2 FOR UPDATE`, member, batch).Scan(&old); e != nil {
			return e
		}
		if old == status {
			return nil
		}
		if status != "ACTIVE" {
			if e := protectRep(ctx, tx, batch, member); e != nil {
				return e
			}
		} else {
			var active bool
			if e := tx.QueryRow(ctx, `SELECT u.status='ACTIVE' FROM users u JOIN batch_memberships m ON m.user_id=u.id WHERE m.id=$1`, member).Scan(&active); e != nil {
				return e
			}
			if !active {
				return apperror.ErrConflict
			}
			if _, e := tx.Exec(ctx, `INSERT INTO membership_roles(membership_id,role_id,assigned_by) SELECT $1,id,$2 FROM roles WHERE code='STUDENT' ON CONFLICT DO NOTHING`, member, actor); e != nil {
				return e
			}
		}
		if _, e := tx.Exec(ctx, `UPDATE batch_memberships SET status=$3,ended_at=CASE WHEN $3 IN ('LEFT','GRADUATED') THEN now() ELSE NULL END,updated_at=now() WHERE id=$1 AND batch_id=$2`, member, batch, status); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, actor, "MEMBERSHIP_"+status, "membership", member, map[string]string{"previous": old, "status": status})
	})
}
func (s Service) List(ctx context.Context, actor, batch, id string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, actor, batch, "membership.view"); e != nil {
		return nil, e
	}
	var b []byte
	e := s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT m.id,m.user_id,m.status,m.joined_at,u.display_name,COALESCE((SELECT json_agg(r.code ORDER BY r.code) FROM membership_roles mr JOIN roles r ON r.id=mr.role_id WHERE mr.membership_id=m.id),'[]') AS roles FROM batch_memberships m JOIN users u ON u.id=m.user_id WHERE m.batch_id=$1 AND ($2='' OR m.id=NULLIF($2,'')::uuid) ORDER BY m.joined_at,m.id LIMIT $3 OFFSET $4) t`, batch, id, limit, offset).Scan(&b)
	if id != "" {
		return db.One(b, e)
	}
	return b, e
}

func (s Service) Catalog(ctx context.Context, user, batch string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "membership.view"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT code,name FROM roles WHERE scope='BATCH' ORDER BY code)t`))
}
