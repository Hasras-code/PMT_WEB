package authorization

import (
	"context"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Authorizer struct{ Pool *pgxpool.Pool }

func Can(ctx context.Context, q Querier, user, batch, permission string) (bool, error) {
	var ok bool
	e := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM batch_memberships m JOIN users u ON u.id=m.user_id JOIN batches b ON b.id=m.batch_id JOIN membership_roles mr ON mr.membership_id=m.id JOIN role_permissions rp ON rp.role_id=mr.role_id JOIN permissions p ON p.id=rp.permission_id WHERE m.user_id=$1 AND m.batch_id=$2 AND m.status='ACTIVE' AND u.status='ACTIVE' AND b.status<>'ARCHIVED' AND p.code=$3 AND p.scope='BATCH')`, user, batch, permission).Scan(&ok)
	return ok, e
}
func Platform(ctx context.Context, q Querier, user, permission string) (bool, error) {
	var ok bool
	e := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_platform_roles ur JOIN users u ON u.id=ur.user_id JOIN role_permissions rp ON rp.role_id=ur.role_id JOIN permissions p ON p.id=rp.permission_id WHERE ur.user_id=$1 AND u.status='ACTIVE' AND p.code=$2 AND p.scope='PLATFORM')`, user, permission).Scan(&ok)
	return ok, e
}
func Require(ctx context.Context, q Querier, user, batch, permission string) error {
	ok, e := Can(ctx, q, user, batch, permission)
	if e != nil {
		return e
	}
	if !ok {
		return apperror.ErrForbidden
	}
	return nil
}
func RequirePlatform(ctx context.Context, q Querier, user, permission string) error {
	ok, e := Platform(ctx, q, user, permission)
	if e != nil {
		return e
	}
	if !ok {
		return apperror.ErrForbidden
	}
	return nil
}

func CanWithPlatform(ctx context.Context, q Querier, user, batch, batchPermission, platformPermission string) (bool, error) {
	platform, e := Platform(ctx, q, user, platformPermission)
	if e != nil || platform {
		return platform, e
	}
	return Can(ctx, q, user, batch, batchPermission)
}

func RequireWithPlatform(ctx context.Context, q Querier, user, batch, batchPermission, platformPermission string) error {
	ok, e := CanWithPlatform(ctx, q, user, batch, batchPermission, platformPermission)
	if e != nil {
		return e
	}
	if !ok {
		return apperror.ErrForbidden
	}
	return nil
}

func WriteWithPlatform(ctx context.Context, tx pgx.Tx, user, batch, batchPermission, platformPermission string) error {
	var id string
	if e := tx.QueryRow(ctx, `SELECT id FROM batches WHERE id=$1 FOR UPDATE`, batch).Scan(&id); e != nil {
		return db.Error(e)
	}
	return RequireWithPlatform(ctx, tx, user, batch, batchPermission, platformPermission)
}

// Batch write operations share this lock with role and membership mutations.
func Write(ctx context.Context, tx pgx.Tx, user, batch, permission string) error {
	var id string
	if e := tx.QueryRow(ctx, `SELECT id FROM batches WHERE id=$1 FOR UPDATE`, batch).Scan(&id); e != nil {
		return db.Error(e)
	}
	return Require(ctx, tx, user, batch, permission)
}
