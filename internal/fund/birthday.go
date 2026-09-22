package fund

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
)

func (s Service) BirthdaySummary(ctx context.Context, user, batch string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(x) FROM(SELECT f.id fund_id,f.name,f.currency,`+balanceSQL+` balance_minor,COALESCE((SELECT sum(c.expected_minor) FROM birthday_contributions c WHERE c.batch_id=f.batch_id),0) total_expected_minor,COALESCE((SELECT sum(p.amount_minor) FROM birthday_contribution_payments p WHERE p.batch_id=f.batch_id),0) total_collected_minor,COALESCE((SELECT sum(c.expected_minor) FROM birthday_contributions c WHERE c.batch_id=f.batch_id),0)-COALESCE((SELECT sum(p.amount_minor) FROM birthday_contribution_payments p WHERE p.batch_id=f.batch_id),0) outstanding_minor FROM funds f LEFT JOIN fund_transactions t ON t.fund_id=f.id WHERE f.batch_id=$1 AND f.type='BIRTHDAY' GROUP BY f.id)x`, batch))
}

func (s Service) Periods(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT p.id,p.year,p.month,p.amount_minor,p.status,p.created_at,p.closed_at,count(c.id) member_count,COALESCE(sum(c.expected_minor),0) expected_minor,COALESCE((SELECT sum(bp.amount_minor) FROM birthday_contribution_payments bp JOIN birthday_contributions bc ON bc.id=bp.contribution_id WHERE bc.period_id=p.id),0) collected_minor FROM birthday_contribution_periods p LEFT JOIN birthday_contributions c ON c.period_id=p.id WHERE p.batch_id=$1 GROUP BY p.id ORDER BY p.year DESC,p.month DESC LIMIT $2 OFFSET $3)x`, batch, limit, offset))
}

func (s Service) CreatePeriod(ctx context.Context, user, batch string, in PeriodInput) (string, error) {
	if in.Year < 2000 || in.Year > 2200 || in.Month < 1 || in.Month > 12 || in.AmountMinor <= 0 {
		return "", apperror.ErrInvalid
	}
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "birthday_fund.manage"); err != nil {
			return err
		}
		var fundID, status string
		if err := tx.QueryRow(ctx, `SELECT id,status FROM funds WHERE batch_id=$1 AND type='BIRTHDAY' FOR UPDATE`, batch).Scan(&fundID, &status); err != nil {
			return db.Error(err)
		}
		if status != "ACTIVE" {
			return ErrClosed
		}
		if err := tx.QueryRow(ctx, `INSERT INTO birthday_contribution_periods(batch_id,fund_id,year,month,amount_minor,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, batch, fundID, in.Year, in.Month, in.AmountMinor, user).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO birthday_contributions(batch_id,period_id,membership_id,expected_minor) SELECT $1,$2,m.id,$3 FROM batch_memberships m WHERE m.batch_id=$1 AND m.status='ACTIVE' AND EXISTS(SELECT 1 FROM membership_roles mr JOIN roles r ON r.id=mr.role_id WHERE mr.membership_id=m.id AND r.code='STUDENT' AND r.scope='BATCH')`, batch, id, in.AmountMinor); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "BIRTHDAY_PERIOD_CREATED", "birthday_contribution_period", id, map[string]any{"year": in.Year, "month": in.Month, "amount_minor": in.AmountMinor})
	})
	return id, err
}

func (s Service) Period(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(x) FROM(SELECT p.id,p.year,p.month,p.amount_minor,p.status,p.created_at,p.closed_at,count(c.id) member_count,COALESCE(sum(c.expected_minor),0) expected_minor,COALESCE((SELECT sum(bp.amount_minor) FROM birthday_contribution_payments bp JOIN birthday_contributions bc ON bc.id=bp.contribution_id WHERE bc.period_id=p.id),0) collected_minor FROM birthday_contribution_periods p LEFT JOIN birthday_contributions c ON c.period_id=p.id WHERE p.batch_id=$1 AND p.id=$2 GROUP BY p.id)x`, batch, id))
}

func (s Service) ClosePeriod(ctx context.Context, user, batch, id string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "birthday_fund.manage"); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE birthday_contribution_periods SET status='CLOSED',closed_at=now(),closed_by=$3 WHERE batch_id=$1 AND id=$2 AND status='OPEN'`, batch, id, user)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrPeriodClosed
		}
		return audit.Record(ctx, tx, batch, user, "BIRTHDAY_PERIOD_CLOSED", "birthday_contribution_period", id, nil)
	})
}

func (s Service) Contributions(ctx context.Context, user, batch, period string, limit, offset int) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "birthday_fund.manage"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT c.id,c.membership_id,m.user_id,u.display_name,c.expected_minor,COALESCE(sum(p.amount_minor),0) paid_minor,c.expected_minor-COALESCE(sum(p.amount_minor),0) outstanding_minor,CASE WHEN COALESCE(sum(p.amount_minor),0)>=c.expected_minor THEN 'PAID' WHEN COALESCE(sum(p.amount_minor),0)>0 THEN 'PARTIAL' ELSE 'UNPAID' END status FROM birthday_contributions c JOIN batch_memberships m ON m.id=c.membership_id AND m.batch_id=c.batch_id JOIN users u ON u.id=m.user_id LEFT JOIN birthday_contribution_payments p ON p.contribution_id=c.id WHERE c.batch_id=$1 AND c.period_id=$2 GROUP BY c.id,m.user_id,u.display_name ORDER BY u.display_name,c.id LIMIT $3 OFFSET $4)x`, batch, period, limit, offset))
}

func (s Service) MyContribution(ctx context.Context, user, batch, period string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(x) FROM(SELECT c.id,c.expected_minor,COALESCE(sum(p.amount_minor),0) paid_minor,c.expected_minor-COALESCE(sum(p.amount_minor),0) outstanding_minor,CASE WHEN COALESCE(sum(p.amount_minor),0)>=c.expected_minor THEN 'PAID' WHEN COALESCE(sum(p.amount_minor),0)>0 THEN 'PARTIAL' ELSE 'UNPAID' END status FROM birthday_contributions c JOIN batch_memberships m ON m.id=c.membership_id AND m.batch_id=c.batch_id LEFT JOIN birthday_contribution_payments p ON p.contribution_id=c.id WHERE c.batch_id=$1 AND c.period_id=$2 AND m.user_id=$3 GROUP BY c.id)x`, batch, period, user))
}

func (s Service) RecordPayment(ctx context.Context, user, batch, period, membership string, in PaymentInput) (string, error) {
	if in.AmountMinor <= 0 || len(in.Description) > 20000 {
		return "", apperror.ErrInvalid
	}
	var paymentID string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := authorization.Write(ctx, tx, user, batch, "birthday_contribution.record"); err != nil {
			return err
		}
		var contribution, fundID, periodStatus, display string
		var expected, paid int64
		if err := tx.QueryRow(ctx, `SELECT c.id,p.fund_id,p.status,c.expected_minor,u.display_name,COALESCE((SELECT sum(bp.amount_minor) FROM birthday_contribution_payments bp WHERE bp.contribution_id=c.id),0) FROM birthday_contributions c JOIN birthday_contribution_periods p ON p.id=c.period_id AND p.batch_id=c.batch_id JOIN batch_memberships m ON m.id=c.membership_id AND m.batch_id=c.batch_id JOIN users u ON u.id=m.user_id WHERE c.batch_id=$1 AND c.period_id=$2 AND c.membership_id=$3 FOR UPDATE OF c,p`, batch, period, membership).Scan(&contribution, &fundID, &periodStatus, &expected, &display, &paid); err != nil {
			return db.Error(err)
		}
		if periodStatus != "OPEN" {
			return ErrPeriodClosed
		}
		if in.AmountMinor > expected-paid {
			return ErrRepayment
		}
		if _, err := lockFund(ctx, tx, batch, fundID); err != nil {
			return err
		}
		description := strings.TrimSpace(in.Description)
		if description == "" {
			description = "Birthday contribution"
		}
		var transaction string
		if err := tx.QueryRow(ctx, `INSERT INTO fund_transactions(batch_id,fund_id,type,amount_minor,description,source_type,source_name,transaction_date,created_by,status,posted_at) VALUES($1,$2,'CASH_IN',$3,$4,'STUDENT_CONTRIBUTION',$5,now(),$6,'POSTED',now()) RETURNING id`, batch, fundID, in.AmountMinor, description, display, user).Scan(&transaction); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO birthday_contribution_payments(batch_id,contribution_id,transaction_id,amount_minor,recorded_by) VALUES($1,$2,$3,$4,$5) RETURNING id`, batch, contribution, transaction, in.AmountMinor, user).Scan(&paymentID); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "BIRTHDAY_PAYMENT_RECORDED", "birthday_contribution_payment", paymentID, map[string]any{"period_id": period, "amount_minor": in.AmountMinor})
	})
	return paymentID, err
}
