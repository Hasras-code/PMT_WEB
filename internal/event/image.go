package event

import (
	"context"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5"
)

func (s Service) SetImage(ctx context.Context, user, batch, id, uploadID string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "event.manage"); e != nil {
			return e
		}
		o, e := upload.Consume(ctx, tx, user, batch, uploadID, "event")
		if e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE events SET cover_image_key=$3,updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, o.Key)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return nil
	})
}
func (s Service) Image(ctx context.Context, user, batch, id string) (upload.Object, error) {
	var o upload.Object
	if _, e := s.Get(ctx, user, batch, id); e != nil {
		return o, e
	}
	e := s.Pool.QueryRow(ctx, `SELECT i.storage_key,i.file_name,i.mime_type FROM events e JOIN upload_intents i ON i.storage_key=e.cover_image_key WHERE e.batch_id=$1 AND e.id=$2`, batch, id).Scan(&o.Key, &o.Name, &o.MIME)
	return o, db.Error(e)
}
