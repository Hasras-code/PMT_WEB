package user

import (
	"context"
	"encoding/json"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ Pool *pgxpool.Pool }
type Profile struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	DisplayName *string `json:"display_name"`
	Phone       *string `json:"phone_number"`
	UploadID    string  `json:"upload_id"`
}

func (s Service) Update(ctx context.Context, user string, in Profile) error {
	for _, v := range []*string{in.FirstName, in.LastName, in.DisplayName} {
		if v != nil && (*v == "" || len(*v) > 100) {
			return apperror.ErrInvalid
		}
	}
	if in.Phone != nil && len(*in.Phone) > 40 {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var key *string
		if in.UploadID != "" {
			o, e := upload.Consume(ctx, tx, user, "", in.UploadID, "profile")
			if e != nil {
				return e
			}
			key = &o.Key
		}
		_, e := tx.Exec(ctx, `UPDATE users SET first_name=COALESCE($2,first_name),last_name=COALESCE($3,last_name),display_name=COALESCE($4,display_name),phone_number=COALESCE($5,phone_number),profile_image_key=COALESCE($6,profile_image_key),updated_at=now() WHERE id=$1`, user, in.FirstName, in.LastName, in.DisplayName, in.Phone, key)
		return e
	})
}
func (s Service) Sessions(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,created_at,last_used_at,expires_at,user_agent FROM auth_sessions WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>now() ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3)t`, user, limit, offset))
}
func (s Service) Bookmarks(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT r.id,r.batch_id,r.title,r.type,b.created_at FROM bookmarks b JOIN resources r ON r.id=b.resource_id AND r.batch_id=b.batch_id JOIN batch_memberships m ON m.batch_id=r.batch_id AND m.user_id=b.user_id JOIN batches cohort ON cohort.id=r.batch_id WHERE b.user_id=$1 AND m.status='ACTIVE' AND cohort.status<>'ARCHIVED' AND r.status='PUBLISHED' ORDER BY b.created_at DESC,r.id DESC LIMIT $2 OFFSET $3)t`, user, limit, offset))
}

// Access returns the current platform and batch authorization context used by
// clients to render only actions the user can actually perform. Authorization
// remains enforced independently by every domain operation.
func (s Service) Access(ctx context.Context, user string) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `
		SELECT json_build_object(
			'platform_roles', COALESCE((
				SELECT json_agg(DISTINCT r.code ORDER BY r.code)
				FROM user_platform_roles upr
				JOIN roles r ON r.id=upr.role_id
				WHERE upr.user_id=$1
			), '[]'::json),
			'platform_permissions', COALESCE((
				SELECT json_agg(DISTINCT p.code ORDER BY p.code)
				FROM user_platform_roles upr
				JOIN role_permissions rp ON rp.role_id=upr.role_id
				JOIN permissions p ON p.id=rp.permission_id AND p.scope='PLATFORM'
				WHERE upr.user_id=$1
			), '[]'::json),
			'memberships', COALESCE((
				SELECT json_agg(row_to_json(membership) ORDER BY membership.entry_year DESC, membership.batch_id)
				FROM (
					SELECT m.id AS membership_id,m.batch_id,b.name AS batch_name,b.slug AS batch_slug,b.entry_year,m.status,
						COALESCE((SELECT json_agg(DISTINCT r.code ORDER BY r.code) FROM membership_roles mr JOIN roles r ON r.id=mr.role_id WHERE mr.membership_id=m.id),'[]'::json) AS roles,
						COALESCE((SELECT json_agg(DISTINCT p.code ORDER BY p.code) FROM membership_roles mr JOIN role_permissions rp ON rp.role_id=mr.role_id JOIN permissions p ON p.id=rp.permission_id AND p.scope='BATCH' WHERE mr.membership_id=m.id),'[]'::json) AS permissions
					FROM batch_memberships m
					JOIN batches b ON b.id=m.batch_id
					WHERE m.user_id=$1 AND m.status='ACTIVE' AND b.status<>'ARCHIVED'
				) membership
			), '[]'::json)
		)`, user))
}
func (s Service) AdminList(ctx context.Context, actor, id string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.RequirePlatform(ctx, s.Pool, actor, "platform_user.manage"); e != nil {
		return nil, e
	}
	b, e := db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT u.id,u.student_number,u.combination,u.first_name,u.last_name,u.display_name,u.email,u.status,u.created_at,COALESCE((SELECT json_agg(r.code ORDER BY r.code) FROM user_platform_roles upr JOIN roles r ON r.id=upr.role_id WHERE upr.user_id=u.id AND upr.scope='PLATFORM'),'[]') AS platform_roles,COALESCE((SELECT json_agg(json_build_object('batch_id',m.batch_id,'batch_name',b.name,'code',r.code) ORDER BY b.entry_year DESC,r.code) FROM batch_memberships m JOIN batches b ON b.id=m.batch_id JOIN membership_roles mr ON mr.membership_id=m.id JOIN roles r ON r.id=mr.role_id WHERE m.user_id=u.id AND m.status='ACTIVE' AND r.code<>'STUDENT'),'[]') AS batch_roles FROM users u WHERE ($1='' OR u.id=NULLIF($1,'')::uuid) ORDER BY u.created_at DESC,u.id DESC LIMIT $2 OFFSET $3)t`, id, limit, offset))
	if id != "" {
		return db.One(b, e)
	}
	return b, e
}

func (s Service) AdminStats(ctx context.Context, actor string) (json.RawMessage, error) {
	if e := authorization.RequirePlatform(ctx, s.Pool, actor, "platform_user.manage"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT json_build_object('users',count(*)) FROM users`))
}

func (s Service) PlatformRoles(ctx context.Context, actor string) (json.RawMessage, error) {
	if e := authorization.RequirePlatform(ctx, s.Pool, actor, "platform_user.manage"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT code,name,scope FROM roles ORDER BY scope,code)t`))
}

func (s Service) AdminBatches(ctx context.Context, actor string) (json.RawMessage, error) {
	if e := authorization.RequirePlatform(ctx, s.Pool, actor, "platform_user.manage"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,name,slug,entry_year FROM batches WHERE status<>'ARCHIVED' ORDER BY entry_year DESC,id DESC)t`))
}

func (s Service) Role(ctx context.Context, actor, target, role, batchID string, remove bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.RequirePlatform(ctx, tx, actor, "platform_user.manage"); e != nil {
			return e
		}
		var roleID, scope string
		if e := tx.QueryRow(ctx, `SELECT id,scope FROM roles WHERE code=$1`, role).Scan(&roleID, &scope); e != nil {
			return apperror.ErrInvalid
		}
		var exists bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, target).Scan(&exists); e != nil {
			return e
		}
		if !exists {
			return apperror.ErrNotFound
		}
		if scope == "PLATFORM" {
			if actor == target {
				return apperror.ErrConflict
			}
			if batchID != "" {
				return apperror.ErrInvalid
			}
			if remove {
				_, e := tx.Exec(ctx, `DELETE FROM user_platform_roles WHERE user_id=$1 AND role_id=$2`, target, roleID)
				if e != nil {
					return e
				}
			} else {
				_, e := tx.Exec(ctx, `INSERT INTO user_platform_roles(user_id,role_id,scope,assigned_by) VALUES($1,$2,'PLATFORM',$3) ON CONFLICT DO NOTHING`, target, roleID, actor)
				if e != nil {
					return e
				}
			}
		} else {
			if batchID == "" {
				return apperror.ErrInvalid
			}
			var membershipID string
			if e := tx.QueryRow(ctx, `SELECT id FROM batch_memberships WHERE batch_id=$1 AND user_id=$2`, batchID, target).Scan(&membershipID); e != nil {
				if e != pgx.ErrNoRows {
					return e
				}
				if e := tx.QueryRow(ctx, `INSERT INTO batch_memberships(batch_id,user_id) VALUES($1,$2) RETURNING id`, batchID, target).Scan(&membershipID); e != nil {
					return e
				}
				if _, e := tx.Exec(ctx, `INSERT INTO membership_roles(membership_id,role_id,assigned_by) SELECT $1,id,$2 FROM roles WHERE code='STUDENT' AND scope='BATCH' ON CONFLICT DO NOTHING`, membershipID, actor); e != nil {
					return e
				}
			}
			if remove {
				_, e := tx.Exec(ctx, `DELETE FROM membership_roles WHERE membership_id=$1 AND role_id=$2`, membershipID, roleID)
				if e != nil {
					return e
				}
			} else {
				_, e := tx.Exec(ctx, `INSERT INTO membership_roles(membership_id,role_id,scope,assigned_by) VALUES($1,$2,'BATCH',$3) ON CONFLICT DO NOTHING`, membershipID, roleID, actor)
				if e != nil {
					return e
				}
			}
		}
		action := map[bool]string{true: "ROLE_REMOVED", false: "ROLE_ASSIGNED"}[remove]
		auditBatchID := batchID
		if scope == "PLATFORM" {
			action = map[bool]string{true: "PLATFORM_ROLE_REMOVED", false: "PLATFORM_ROLE_ASSIGNED"}[remove]
			auditBatchID = ""
		}
		return audit.Record(ctx, tx, auditBatchID, actor, action, "user", target, map[string]string{"role": role})
	})
}
func (s Service) Status(ctx context.Context, actor, id, status string) error {
	switch status {
	case "ACTIVE", "SUSPENDED", "ARCHIVED":
	default:
		return apperror.ErrInvalid
	}
	if actor == id {
		return apperror.ErrConflict
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.RequirePlatform(ctx, tx, actor, "platform_user.manage"); e != nil {
			return e
		}
		var old string
		var verified bool
		if e := tx.QueryRow(ctx, `SELECT status,email_verified_at IS NOT NULL FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&old, &verified); e != nil {
			return e
		}
		if old == "ARCHIVED" || (!verified && status == "ACTIVE") {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE users SET status=$2,updated_at=now() WHERE id=$1`, id, status); e != nil {
			return e
		}
		if status != "ACTIVE" {
			if _, e := tx.Exec(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`, id); e != nil {
				return e
			}
		}
		return audit.Record(ctx, tx, "", actor, "USER_"+status, "user", id, nil)
	})
}
