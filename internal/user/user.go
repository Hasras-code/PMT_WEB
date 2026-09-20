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
func (s Service) AdminList(ctx context.Context, actor, id string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.RequirePlatform(ctx, s.Pool, actor, "platform_user.manage"); e != nil {
		return nil, e
	}
	b, e := db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,student_number,first_name,last_name,display_name,email,status,created_at FROM users WHERE ($1='' OR id=NULLIF($1,'')::uuid) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3)t`, id, limit, offset))
	if id != "" {
		return db.One(b, e)
	}
	return b, e
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
