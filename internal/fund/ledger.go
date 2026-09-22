package fund

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
)

func validateTransaction(in TransactionInput) error {
	if in.Type != "CASH_IN" && in.Type != "EXPENSE" || in.AmountMinor <= 0 || len(in.Description) > 20000 || in.SourceType != nil && *in.SourceType == "STUDENT_CONTRIBUTION" {
		return apperror.ErrInvalid
	}
	for _, v := range []*string{in.Category, in.SourceType, in.SourceName, in.Reference} {
		if v != nil && len(*v) > 300 {
			return apperror.ErrInvalid
		}
	}
	if in.Status == "" {
		in.Status = "POSTED"
	}
	if in.Status != "DRAFT" && in.Status != "POSTED" {
		return apperror.ErrInvalid
	}
	return nil
}

func (s Service) CreateTransaction(ctx context.Context, user, batch, fundID string, in TransactionInput) (string, error) {
	if in.Status == "" {
		in.Status = "POSTED"
	}
	if err := validateTransaction(in); err != nil {
		return "", err
	}
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		status, err := lockFund(ctx, tx, batch, fundID)
		if err != nil {
			return err
		}
		if status != "ACTIVE" {
			return ErrClosed
		}
		if err = requireManage(ctx, tx, user, batch, fundID, "fund.transaction.create"); err != nil {
			return err
		}
		if in.Status == "POSTED" && in.Type == "EXPENSE" {
			available, err := balance(ctx, tx, fundID)
			if err != nil {
				return err
			}
			if available < in.AmountMinor {
				return ErrInsufficient
			}
		}
		date := time.Now()
		if in.TransactionDate != nil {
			date = *in.TransactionDate
		}
		err = tx.QueryRow(ctx, `INSERT INTO fund_transactions(batch_id,fund_id,type,amount_minor,description,category,source_type,source_name,reference,transaction_date,created_by,status,posted_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,CASE WHEN $12='POSTED' THEN now() END) RETURNING id`, batch, fundID, in.Type, in.AmountMinor, in.Description, in.Category, in.SourceType, in.SourceName, in.Reference, date, user, in.Status).Scan(&id)
		if err != nil {
			return err
		}
		if in.Status == "POSTED" {
			return audit.Record(ctx, tx, batch, user, "FUND_TRANSACTION_POSTED", "fund_transaction", id, map[string]any{"fund_id": fundID, "type": in.Type, "amount_minor": in.AmountMinor})
		}
		return nil
	})
	return id, err
}

func (s Service) UpdateTransaction(ctx context.Context, user, batch, fundID, id string, in TransactionUpdate) error {
	if in.AmountMinor != nil && *in.AmountMinor <= 0 || in.Description != nil && len(*in.Description) > 20000 {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := lockFund(ctx, tx, batch, fundID); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, fundID, "fund.transaction.create"); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE fund_transactions SET amount_minor=COALESCE($4,amount_minor),description=COALESCE($5,description),category=COALESCE($6,category),source_type=COALESCE($7,source_type),source_name=COALESCE($8,source_name),reference=COALESCE($9,reference),transaction_date=COALESCE($10,transaction_date) WHERE batch_id=$1 AND fund_id=$2 AND id=$3 AND status='DRAFT' AND transfer_id IS NULL`, batch, fundID, id, in.AmountMinor, in.Description, in.Category, in.SourceType, in.SourceName, in.Reference, in.TransactionDate)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrConflict
		}
		return nil
	})
}

func (s Service) PostTransaction(ctx context.Context, user, batch, fundID, id string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		status, err := lockFund(ctx, tx, batch, fundID)
		if err != nil {
			return err
		}
		if status != "ACTIVE" {
			return ErrClosed
		}
		if err = requireManage(ctx, tx, user, batch, fundID, "fund.transaction.create"); err != nil {
			return err
		}
		var typ string
		var amount int64
		err = tx.QueryRow(ctx, `SELECT type,amount_minor FROM fund_transactions WHERE batch_id=$1 AND fund_id=$2 AND id=$3 AND status='DRAFT' AND transfer_id IS NULL FOR UPDATE`, batch, fundID, id).Scan(&typ, &amount)
		if err != nil {
			return db.Error(err)
		}
		if typ == "EXPENSE" {
			available, err := balance(ctx, tx, fundID)
			if err != nil {
				return err
			}
			if available < amount {
				return ErrInsufficient
			}
		}
		_, err = tx.Exec(ctx, `UPDATE fund_transactions SET status='POSTED',posted_at=now() WHERE id=$1`, id)
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_TRANSACTION_POSTED", "fund_transaction", id, map[string]any{"fund_id": fundID, "type": typ, "amount_minor": amount})
	})
}

func (s Service) Transactions(ctx context.Context, user, batch, fundID, typ, source, dateFrom, dateTo string, includeDraft bool, limit, offset int) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	if err := ensureFund(ctx, s.Pool, batch, fundID); err != nil {
		return nil, err
	}
	manage, err := canManage(ctx, s.Pool, user, batch, fundID, "fund.transaction.create")
	if err != nil {
		return nil, err
	}
	if includeDraft && !manage {
		return nil, apperror.ErrForbidden
	}
	allowed := map[string]bool{"": true, "CASH_IN": true, "EXPENSE": true, "TRANSFER_IN": true, "TRANSFER_OUT": true, "REVERSAL_IN": true, "REVERSAL_OUT": true}
	if !allowed[typ] {
		return nil, apperror.ErrInvalid
	}
	var from, to *time.Time
	if dateFrom != "" {
		v, e := time.Parse(time.RFC3339, dateFrom)
		if e != nil {
			return nil, apperror.ErrInvalid
		}
		from = &v
	}
	if dateTo != "" {
		v, e := time.Parse(time.RFC3339, dateTo)
		if e != nil {
			return nil, apperror.ErrInvalid
		}
		to = &v
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, apperror.ErrInvalid
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT id,fund_id,type,amount_minor,description,category,source_type,CASE WHEN source_type='STUDENT_CONTRIBUTION' AND NOT $3 THEN NULL ELSE source_name END source_name,reference,transaction_date,status,transfer_id,reversal_of_transaction_id,reversed_by_transaction_id,created_at,posted_at FROM fund_transactions WHERE batch_id=$1 AND fund_id=$2 AND (status='POSTED' OR $3) AND ($4='' OR type=$4) AND ($5='' OR source_type=$5) AND ($6::timestamptz IS NULL OR transaction_date >= $6) AND ($7::timestamptz IS NULL OR transaction_date <= $7) ORDER BY transaction_date DESC,id DESC LIMIT $8 OFFSET $9)x`, batch, fundID, includeDraft && manage, typ, source, from, to, limit, offset))
}

func (s Service) Transaction(ctx context.Context, user, batch, fundID, id string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	manage, err := canManage(ctx, s.Pool, user, batch, fundID, "fund.transaction.create")
	if err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(x) FROM(SELECT id,fund_id,type,amount_minor,description,category,source_type,CASE WHEN source_type='STUDENT_CONTRIBUTION' AND NOT $4 THEN NULL ELSE source_name END source_name,reference,transaction_date,status,transfer_id,reversal_of_transaction_id,reversed_by_transaction_id,created_at,posted_at FROM fund_transactions WHERE batch_id=$1 AND fund_id=$2 AND id=$3 AND (status='POSTED' OR $4))x`, batch, fundID, id, manage))
}

func reversalType(t string) (string, bool) {
	switch t {
	case "CASH_IN":
		return "REVERSAL_OUT", true
	case "EXPENSE":
		return "REVERSAL_IN", true
	default:
		return "", false
	}
}
func (s Service) ReverseTransaction(ctx context.Context, user, batch, fundID, id string) (string, error) {
	var reversal string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		_, err := lockFund(ctx, tx, batch, fundID)
		if err != nil {
			return err
		}
		if err = requireManage(ctx, tx, user, batch, fundID, "fund.transaction.reverse"); err != nil {
			return err
		}
		var typ, description string
		var amount int64
		var reversed *string
		err = tx.QueryRow(ctx, `SELECT type,amount_minor,description,reversed_by_transaction_id FROM fund_transactions WHERE batch_id=$1 AND fund_id=$2 AND id=$3 AND status='POSTED' AND transfer_id IS NULL FOR UPDATE`, batch, fundID, id).Scan(&typ, &amount, &description, &reversed)
		if err != nil {
			return db.Error(err)
		}
		if reversed != nil {
			return ErrReversed
		}
		rt, ok := reversalType(typ)
		if !ok {
			return apperror.ErrConflict
		}
		if rt == "REVERSAL_OUT" {
			available, err := balance(ctx, tx, fundID)
			if err != nil {
				return err
			}
			if available < amount {
				return ErrInsufficient
			}
		}
		err = tx.QueryRow(ctx, `INSERT INTO fund_transactions(batch_id,fund_id,type,amount_minor,description,transaction_date,created_by,status,posted_at,reversal_of_transaction_id) VALUES($1,$2,$3,$4,$5,now(),$6,'POSTED',now(),$7) RETURNING id`, batch, fundID, rt, amount, "Reversal: "+strings.TrimSpace(description), user, id).Scan(&reversal)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE fund_transactions SET reversed_by_transaction_id=$2 WHERE id=$1`, id, reversal); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_TRANSACTION_REVERSED", "fund_transaction", id, map[string]any{"reversal_id": reversal})
	})
	return reversal, err
}
