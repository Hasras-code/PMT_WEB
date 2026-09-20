package announcement

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5"
)

func (s Service) Attach(ctx context.Context, user, batch, id, uploadID string) (string, error) {
	var aid string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "announcement.update"); e != nil {
			return e
		}
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM announcements WHERE batch_id=$1 AND id=$2`, batch, id).Scan(&status); e != nil {
			return e
		}
		if status == "ARCHIVED" {
			return apperror.ErrConflict
		}
		if status == "PUBLISHED" {
			if e := authorization.Require(ctx, tx, user, batch, "announcement.publish"); e != nil {
				return e
			}
		}
		o, e := upload.Consume(ctx, tx, user, batch, uploadID, "attachment")
		if e != nil {
			return e
		}
		return tx.QueryRow(ctx, `INSERT INTO announcement_attachments(batch_id,announcement_id,file_name,storage_key,mime_type,size_bytes) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, batch, id, o.Name, o.Key, o.MIME, o.Size).Scan(&aid)
	})
	return aid, e
}
func (s Service) Attachment(ctx context.Context, user, batch, id, aid string) (upload.Object, error) {
	var o upload.Object
	if _, e := s.Get(ctx, user, batch, id); e != nil {
		return o, e
	}
	e := s.Pool.QueryRow(ctx, `SELECT storage_key,file_name,mime_type,size_bytes FROM announcement_attachments WHERE batch_id=$1 AND announcement_id=$2 AND id=$3`, batch, id, aid).Scan(&o.Key, &o.Name, &o.MIME, &o.Size)
	return o, db.Error(e)
}
func (s Service) RemoveAttachment(ctx context.Context, user, batch, id, aid string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "announcement.update"); e != nil {
			return e
		}
		var state string
		if e := tx.QueryRow(ctx, `SELECT status FROM announcements WHERE batch_id=$1 AND id=$2`, batch, id).Scan(&state); e != nil {
			return e
		}
		if state == "PUBLISHED" {
			if e := authorization.Require(ctx, tx, user, batch, "announcement.publish"); e != nil {
				return e
			}
		}
		if state == "ARCHIVED" {
			return apperror.ErrConflict
		}
		tag, e := tx.Exec(ctx, `DELETE FROM announcement_attachments WHERE batch_id=$1 AND announcement_id=$2 AND id=$3`, batch, id, aid)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return nil
	})
}

func (s Service) Attachments(ctx context.Context, user, batch, id string, limit, offset int) (json.RawMessage, error) {
	if _, e := s.Get(ctx, user, batch, id); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,file_name,mime_type,size_bytes,created_at FROM announcement_attachments WHERE batch_id=$1 AND announcement_id=$2 ORDER BY created_at,id LIMIT $3 OFFSET $4)t`, batch, id, limit, offset))
}
