package fund

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
)

func lockFunds(ctx context.Context, tx pgx.Tx, batch, a, b string) error {
	ids := []string{a, b}
	sort.Strings(ids)
	for _, id := range ids {
		status, err := lockFund(ctx, tx, batch, id)
		if err != nil {
			return err
		}
		if status != "ACTIVE" {
			return ErrClosed
		}
	}
	return nil
}

func insertTransfer(ctx context.Context, tx pgx.Tx, user, batch, from, to, typ string, amount int64, description, parent, reversalOf string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO fund_transfers(batch_id,from_fund_id,to_fund_id,type,amount_minor,description,created_by,parent_transfer_id,reversal_of_transfer_id) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid) RETURNING id`, batch, from, to, typ, amount, description, user, parent, reversalOf).Scan(&id)
	if err != nil {
		return "", err
	}
	for _, entry := range []struct{ fund, typ string }{{from, "TRANSFER_OUT"}, {to, "TRANSFER_IN"}} {
		if _, err = tx.Exec(ctx, `INSERT INTO fund_transactions(batch_id,fund_id,type,amount_minor,description,transaction_date,created_by,status,posted_at,transfer_id) VALUES($1,$2,$3,$4,$5,now(),$6,'POSTED',now(),$7)`, batch, entry.fund, entry.typ, amount, description, user, id); err != nil {
			return "", err
		}
	}
	return id, nil
}

func (s Service) CreateTransfer(ctx context.Context, user, batch string, in TransferInput) (string, error) {
	if in.FromFundID == in.ToFundID || in.AmountMinor <= 0 || len(in.Description) > 20000 || (in.Type != "TRANSFER" && in.Type != "LOAN") {
		return "", ErrInvalidTransfer
	}
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := lockFunds(ctx, tx, batch, in.FromFundID, in.ToFundID); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, in.FromFundID, "fund.transfer.create"); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, in.ToFundID, "fund.transfer.create"); err != nil {
			return err
		}
		available, err := balance(ctx, tx, in.FromFundID)
		if err != nil {
			return err
		}
		if available < in.AmountMinor {
			return ErrInsufficient
		}
		id, err = insertTransfer(ctx, tx, user, batch, in.FromFundID, in.ToFundID, in.Type, in.AmountMinor, in.Description, "", "")
		if err != nil {
			return err
		}
		action := "FUND_TRANSFER_CREATED"
		if in.Type == "LOAN" {
			action = "FUND_LOAN_CREATED"
		}
		return audit.Record(ctx, tx, batch, user, action, "fund_transfer", id, map[string]any{"from_fund_id": in.FromFundID, "to_fund_id": in.ToFundID, "amount_minor": in.AmountMinor})
	})
	return id, err
}

func (s Service) Repay(ctx context.Context, user, batch, loanID string, in RepaymentInput) (string, error) {
	if in.AmountMinor <= 0 || len(in.Description) > 20000 {
		return "", apperror.ErrInvalid
	}
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var from, to, typ string
		var principal int64
		var reversed *string
		if err := tx.QueryRow(ctx, `SELECT from_fund_id,to_fund_id,type,amount_minor,reversed_by_transfer_id FROM fund_transfers WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, loanID).Scan(&from, &to, &typ, &principal, &reversed); err != nil {
			return db.Error(err)
		}
		if typ != "LOAN" || reversed != nil {
			return ErrInvalidTransfer
		}
		var repaid int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_minor),0) FROM fund_transfers WHERE batch_id=$1 AND parent_transfer_id=$2 AND type='LOAN_REPAYMENT' AND reversed_by_transfer_id IS NULL`, batch, loanID).Scan(&repaid); err != nil {
			return err
		}
		if in.AmountMinor > principal-repaid {
			return ErrRepayment
		}
		if err := lockFunds(ctx, tx, batch, to, from); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, to, "fund.transfer.create"); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, from, "fund.transfer.create"); err != nil {
			return err
		}
		available, err := balance(ctx, tx, to)
		if err != nil {
			return err
		}
		if available < in.AmountMinor {
			return ErrInsufficient
		}
		description := strings.TrimSpace(in.Description)
		if description == "" {
			description = "Loan repayment"
		}
		id, err = insertTransfer(ctx, tx, user, batch, to, from, "LOAN_REPAYMENT", in.AmountMinor, description, loanID, "")
		if err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_LOAN_REPAYMENT_CREATED", "fund_transfer", id, map[string]any{"loan_id": loanID, "amount_minor": in.AmountMinor})
	})
	return id, err
}

func (s Service) ReverseTransfer(ctx context.Context, user, batch, originalID string) (string, error) {
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var from, to, typ, description string
		var amount int64
		var reversed *string
		if err := tx.QueryRow(ctx, `SELECT from_fund_id,to_fund_id,type,amount_minor,description,reversed_by_transfer_id FROM fund_transfers WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, originalID).Scan(&from, &to, &typ, &amount, &description, &reversed); err != nil {
			return db.Error(err)
		}
		if reversed != nil {
			return ErrReversed
		}
		if typ == "REVERSAL" {
			return ErrInvalidTransfer
		}
		if typ == "LOAN" {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fund_transfers WHERE parent_transfer_id=$1 AND type='LOAN_REPAYMENT' AND reversed_by_transfer_id IS NULL)`, originalID).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return apperror.WithCode(apperror.ErrConflict, "loan_has_repayments")
			}
		}
		if err := lockFunds(ctx, tx, batch, to, from); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, to, "fund.transfer.reverse"); err != nil {
			return err
		}
		if err := requireManage(ctx, tx, user, batch, from, "fund.transfer.reverse"); err != nil {
			return err
		}
		available, err := balance(ctx, tx, to)
		if err != nil {
			return err
		}
		if available < amount {
			return ErrInsufficient
		}
		id, err = insertTransfer(ctx, tx, user, batch, to, from, "REVERSAL", amount, "Reversal: "+description, "", originalID)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE fund_transfers SET reversed_by_transfer_id=$2,reversed_at=now(),reversed_by=$3 WHERE id=$1`, originalID, id, user); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_TRANSFER_REVERSED", "fund_transfer", originalID, map[string]any{"reversal_id": id})
	})
	return id, err
}

func (s Service) Transfers(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT t.id,t.from_fund_id,ff.name from_fund_name,t.to_fund_id,tf.name to_fund_name,t.type,t.amount_minor,t.description,t.parent_transfer_id,t.reversal_of_transfer_id,t.reversed_by_transfer_id,t.created_at,CASE WHEN t.type='LOAN' THEN t.amount_minor-COALESCE((SELECT sum(r.amount_minor) FROM fund_transfers r WHERE r.parent_transfer_id=t.id AND r.type='LOAN_REPAYMENT' AND r.reversed_by_transfer_id IS NULL),0) END outstanding_minor FROM fund_transfers t JOIN funds ff ON ff.id=t.from_fund_id AND ff.batch_id=t.batch_id JOIN funds tf ON tf.id=t.to_fund_id AND tf.batch_id=t.batch_id WHERE t.batch_id=$1 ORDER BY t.created_at DESC,t.id DESC LIMIT $2 OFFSET $3)x`, batch, limit, offset))
}

func (s Service) Transfer(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return nil, err
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(x) FROM(SELECT t.id,t.from_fund_id,ff.name from_fund_name,t.to_fund_id,tf.name to_fund_name,t.type,t.amount_minor,t.description,t.parent_transfer_id,t.reversal_of_transfer_id,t.reversed_by_transfer_id,t.created_at,CASE WHEN t.type='LOAN' THEN t.amount_minor-COALESCE((SELECT sum(r.amount_minor) FROM fund_transfers r WHERE r.parent_transfer_id=t.id AND r.type='LOAN_REPAYMENT' AND r.reversed_by_transfer_id IS NULL),0) END outstanding_minor FROM fund_transfers t JOIN funds ff ON ff.id=t.from_fund_id AND ff.batch_id=t.batch_id JOIN funds tf ON tf.id=t.to_fund_id AND tf.batch_id=t.batch_id WHERE t.batch_id=$1 AND t.id=$2)x`, batch, id))
}
