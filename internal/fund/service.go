package fund

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/event"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Pool    *pgxpool.Pool
	Uploads upload.Service
}

func EnsureBirthdayFund(ctx context.Context, tx pgx.Tx, batch, actor string) error {
	_, err := tx.Exec(ctx, `INSERT INTO funds(batch_id,name,type,created_by) VALUES($1,'Birthday Fund','BIRTHDAY',NULLIF($2,'')::uuid) ON CONFLICT (batch_id) WHERE type='BIRTHDAY' DO NOTHING`, batch, actor)
	return err
}

func validateFund(in FundInput) error {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 || len(in.Description) > 20000 || (in.Type != "EVENT" && in.Type != "OTHER") || (in.Type != "EVENT" && (in.EventID != nil || in.Event != nil)) || (in.EventID != nil && in.Event != nil) {
		return apperror.ErrInvalid
	}
	if in.Event != nil {
		return in.Event.Validate(true)
	}
	return nil
}

func (s Service) Create(ctx context.Context, user, batch string, in FundInput) (string, error) {
	if err := validateFund(in); err != nil {
		return "", err
	}
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "fund.create"); err != nil {
			return err
		}
		eventID := stringValue(in.EventID)
		if in.Event != nil {
			var err error
			eventID, err = event.CreateTx(ctx, tx, user, batch, *in.Event)
			if err != nil {
				return err
			}
		}
		if err := tx.QueryRow(ctx, `INSERT INTO funds(batch_id,name,description,type,event_id,created_by) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6) RETURNING id`, batch, strings.TrimSpace(in.Name), in.Description, in.Type, eventID, user).Scan(&id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_CREATED", "fund", id, map[string]any{"type": in.Type, "event_id": eventID})
	})
	return id, err
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

const balanceSQL = `COALESCE(sum(CASE WHEN t.type IN ('CASH_IN','TRANSFER_IN','REVERSAL_IN') THEN t.amount_minor WHEN t.type IN ('EXPENSE','TRANSFER_OUT','REVERSAL_OUT') THEN -t.amount_minor ELSE 0 END) FILTER(WHERE t.status='POSTED'),0)`

func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT f.id,f.batch_id,f.name,f.description,f.type,f.event_id,f.currency,f.status,f.created_at,f.closed_at,`+balanceSQL+` AS balance_minor,
	 COALESCE(sum(t.amount_minor) FILTER(WHERE t.status='POSTED' AND t.type='CASH_IN'),0) AS total_cash_in_minor,
	 COALESCE(sum(t.amount_minor) FILTER(WHERE t.status='POSTED' AND t.type='EXPENSE'),0) AS total_expense_minor,
	 COALESCE(sum(t.amount_minor) FILTER(WHERE t.status='POSTED' AND t.type='TRANSFER_IN'),0) AS total_transfer_in_minor,
	 COALESCE(sum(t.amount_minor) FILTER(WHERE t.status='POSTED' AND t.type='TRANSFER_OUT'),0) AS total_transfer_out_minor,
	 COALESCE((SELECT sum(l.amount_minor-COALESCE((SELECT sum(r.amount_minor) FROM fund_transfers r WHERE r.parent_transfer_id=l.id AND r.type='LOAN_REPAYMENT' AND r.reversed_by_transfer_id IS NULL),0)) FROM fund_transfers l WHERE l.batch_id=f.batch_id AND l.to_fund_id=f.id AND l.type='LOAN' AND l.reversed_by_transfer_id IS NULL),0) AS outstanding_loans_payable_minor
	 FROM funds f LEFT JOIN fund_transactions t ON t.fund_id=f.id WHERE f.batch_id=$1 GROUP BY f.id ORDER BY f.type,f.created_at,f.id LIMIT $2 OFFSET $3)x`, batch, limit, offset))
}

func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(x) FROM(SELECT f.id,f.batch_id,f.name,f.description,f.type,f.event_id,f.currency,f.status,f.created_at,f.updated_at,f.closed_at,`+balanceSQL+` AS balance_minor FROM funds f LEFT JOIN fund_transactions t ON t.fund_id=f.id WHERE f.batch_id=$1 AND f.id=$2 GROUP BY f.id)x`, batch, id))
}

func (s Service) Update(ctx context.Context, user, batch, id string, in FundUpdate) error {
	if in.Name != nil && (strings.TrimSpace(*in.Name) == "" || len(*in.Name) > 200) || in.Description != nil && len(*in.Description) > 20000 {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "fund.update"); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE funds SET name=COALESCE($3,name),description=COALESCE($4,description),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status='ACTIVE'`, batch, id, in.Name, in.Description)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "FUND_UPDATED", "fund", id, nil)
	})
}

func (s Service) Transition(ctx context.Context, user, batch, id, target string) error {
	if target != "CLOSED" && target != "ARCHIVED" {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "fund.close"); err != nil {
			return err
		}
		var old string
		if err := tx.QueryRow(ctx, `SELECT status FROM funds WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); err != nil {
			return err
		}
		if old == target {
			return nil
		}
		if target == "CLOSED" && old != "ACTIVE" || target == "ARCHIVED" && old != "CLOSED" {
			return apperror.ErrConflict
		}
		_, err := tx.Exec(ctx, `UPDATE funds SET status=$3,closed_at=COALESCE(closed_at,now()),closed_by=COALESCE(closed_by,$4),updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, target, user)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_"+target, "fund", id, nil)
	})
}

func canManage(ctx context.Context, q authorization.Querier, user, batch, fundID, permission string) (bool, error) {
	ok, err := authorization.Can(ctx, q, user, batch, permission)
	if err != nil || ok {
		return ok, err
	}
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fund_managers fm JOIN batch_memberships m ON m.id=fm.membership_id AND m.batch_id=fm.batch_id JOIN users u ON u.id=m.user_id JOIN funds f ON f.id=fm.fund_id AND f.batch_id=fm.batch_id WHERE fm.batch_id=$1 AND fm.fund_id=$2 AND m.user_id=$3 AND fm.revoked_at IS NULL AND m.status='ACTIVE' AND u.status='ACTIVE' AND f.status<>'ARCHIVED')`, batch, fundID, user).Scan(&ok)
	return ok, err
}
func requireManage(ctx context.Context, q authorization.Querier, user, batch, fundID, permission string) error {
	ok, err := canManage(ctx, q, user, batch, fundID, permission)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.ErrForbidden
	}
	return nil
}

func lockFund(ctx context.Context, tx pgx.Tx, batch, id string) (string, error) {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM funds WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&status)
	return status, db.Error(err)
}
func ensureFund(ctx context.Context, q authorization.Querier, batch, id string) error {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM funds WHERE batch_id=$1 AND id=$2)`, batch, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return apperror.ErrNotFound
	}
	return nil
}
func balance(ctx context.Context, q authorization.Querier, fundID string) (int64, error) {
	var v int64
	err := q.QueryRow(ctx, `SELECT `+balanceSQL+` FROM fund_transactions t WHERE t.fund_id=$1`, fundID).Scan(&v)
	return v, err
}
