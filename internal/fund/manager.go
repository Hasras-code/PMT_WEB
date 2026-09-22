package fund

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
)

func (s Service) Managers(ctx context.Context, user, batch, fundID string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	if err := ensureFund(ctx, s.Pool, batch, fundID); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT fm.id,fm.membership_id,m.user_id,u.display_name,fm.assigned_at FROM fund_managers fm JOIN batch_memberships m ON m.id=fm.membership_id AND m.batch_id=fm.batch_id JOIN users u ON u.id=m.user_id WHERE fm.batch_id=$1 AND fm.fund_id=$2 AND fm.revoked_at IS NULL ORDER BY fm.assigned_at,fm.id)x`, batch, fundID))
}

func (s Service) AssignManager(ctx context.Context, user, batch, fundID, membershipID string) (string, error) {
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "fund.manager.assign"); err != nil {
			return err
		}
		if _, err := lockFund(ctx, tx, batch, fundID); err != nil {
			return err
		}
		var active bool
		if err := tx.QueryRow(ctx, `SELECT status='ACTIVE' FROM batch_memberships WHERE batch_id=$1 AND id=$2`, batch, membershipID).Scan(&active); err != nil {
			return db.Error(err)
		}
		if !active {
			return apperror.ErrConflict
		}
		err := tx.QueryRow(ctx, `INSERT INTO fund_managers(batch_id,fund_id,membership_id,assigned_by) VALUES($1,$2,$3,$4) RETURNING id`, batch, fundID, membershipID, user).Scan(&id)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_MANAGER_ASSIGNED", "fund", fundID, map[string]any{"membership_id": membershipID})
	})
	return id, err
}

func (s Service) RemoveManager(ctx context.Context, user, batch, fundID, membershipID string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "fund.manager.remove"); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE fund_managers SET revoked_at=now(),revoked_by=$4 WHERE batch_id=$1 AND fund_id=$2 AND membership_id=$3 AND revoked_at IS NULL`, batch, fundID, membershipID, user)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "FUND_MANAGER_REMOVED", "fund", fundID, map[string]any{"membership_id": membershipID})
	})
}
