package fund

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5"
)

func (s Service) AuthorizeReceipt(ctx context.Context, user, batch, fundID string, in upload.Input) (json.RawMessage, error) {
	if err := requireManage(ctx, s.Pool, user, batch, fundID, "fund.transaction.create"); err != nil {
		return nil, err
	}
	return s.Uploads.AuthorizeChecked(ctx, user, batch, "fund_receipt", in)
}

func (s Service) Attach(ctx context.Context, user, batch, fundID, transactionID string, in AttachmentInput) (string, error) {
	if in.Visibility == "" {
		in.Visibility = "MANAGERS"
	}
	if in.Visibility != "MEMBERS" && in.Visibility != "MANAGERS" {
		return "", apperror.ErrInvalid
	}
	var id string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := requireManage(ctx, tx, user, batch, fundID, "fund.transaction.create"); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fund_transactions WHERE batch_id=$1 AND fund_id=$2 AND id=$3)`, batch, fundID, transactionID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return apperror.ErrNotFound
		}
		o, err := upload.Consume(ctx, tx, user, batch, in.UploadID, "fund_receipt")
		if err != nil {
			return err
		}
		if o.MIME != "application/pdf" {
			return apperror.ErrInvalid
		}
		if err := tx.QueryRow(ctx, `INSERT INTO fund_transaction_attachments(batch_id,fund_id,transaction_id,file_name,storage_key,mime_type,size_bytes,uploaded_by,visibility) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, batch, fundID, transactionID, o.Name, o.Key, o.MIME, o.Size, user, in.Visibility).Scan(&id); err != nil {
			return err
		}
		return audit.Record(ctx, tx, batch, user, "FUND_RECEIPT_ATTACHED", "fund_transaction_attachment", id, map[string]any{"transaction_id": transactionID, "visibility": in.Visibility})
	})
	return id, err
}

func (s Service) Attachments(ctx context.Context, user, batch, fundID, transactionID string) (json.RawMessage, error) {
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
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(x),'[]') FROM(SELECT id,file_name,mime_type,size_bytes,visibility,created_at FROM fund_transaction_attachments WHERE batch_id=$1 AND fund_id=$2 AND transaction_id=$3 AND (visibility='MEMBERS' OR $4) ORDER BY created_at,id)x`, batch, fundID, transactionID, manage))
}

func (s Service) Attachment(ctx context.Context, user, batch, fundID, transactionID, id string) (upload.Object, error) {
	var o upload.Object
	if err := authorization.Require(ctx, s.Pool, user, batch, "fund.view"); err != nil {
		return o, err
	}
	manage, err := canManage(ctx, s.Pool, user, batch, fundID, "fund.transaction.create")
	if err != nil {
		return o, err
	}
	err = s.Pool.QueryRow(ctx, `SELECT id,storage_key,file_name,mime_type,size_bytes FROM fund_transaction_attachments WHERE batch_id=$1 AND fund_id=$2 AND transaction_id=$3 AND id=$4 AND (visibility='MEMBERS' OR $5)`, batch, fundID, transactionID, id, manage).Scan(&o.ID, &o.Key, &o.Name, &o.MIME, &o.Size)
	return o, db.Error(err)
}
