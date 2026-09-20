package notification

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ Pool *pgxpool.Pool }

func (s Service) List(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,batch_id,type,title,body,entity_id,read_at,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3)t`, user, limit, offset))
}
func (s Service) Read(ctx context.Context, user, id string) error {
	tag, e := s.Pool.Exec(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE user_id=$1 AND ($2='' OR id=NULLIF($2,'')::uuid)`, user, id)
	if e != nil {
		return e
	}
	if id != "" && tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
func (s Service) Dispatch(ctx context.Context) (int, error) {
	count := 0
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `SELECT id FROM notification_events WHERE processed_at IS NULL ORDER BY created_at LIMIT 50 FOR UPDATE SKIP LOCKED`)
		if e != nil {
			return e
		}
		var ids []string
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		rows.Close()
		if e = rows.Err(); e != nil {
			return e
		}
		for _, id := range ids {
			if _, e = tx.Exec(ctx, `INSERT INTO notifications(user_id,batch_id,event_id,type,title,entity_id) SELECT m.user_id,e.batch_id,e.id,e.event_type,e.title,e.entity_id FROM notification_events e JOIN batch_memberships m ON m.batch_id=e.batch_id JOIN users u ON u.id=m.user_id WHERE e.id=$1 AND m.status='ACTIVE' AND u.status='ACTIVE' ON CONFLICT DO NOTHING`, id); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE notification_events SET processed_at=now() WHERE id=$1`, id); e != nil {
				return e
			}
			count++
		}
		return nil
	})
	return count, e
}
