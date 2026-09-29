package upload

import (
	"context"

	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cleanup only removes expired, unconsumed uploads. Historical content is retained.
func Cleanup(ctx context.Context, p *pgxpool.Pool, store *storage.Manager) (int, error) {
	rows, e := p.Query(ctx, `SELECT id,storage_key,mime_type,size_bytes,storage_provider,storage_class FROM upload_intents WHERE state<>'CONSUMED' AND expires_at<now()-interval '1 hour' ORDER BY expires_at LIMIT 100`)
	if e != nil {
		return 0, e
	}
	type item struct {
		id     string
		object storage.Object
	}
	var items []item
	for rows.Next() {
		var i item
		if e = rows.Scan(&i.id, &i.object.Key, &i.object.MIME, &i.object.Size, &i.object.Provider, &i.object.Class); e != nil {
			rows.Close()
			return 0, e
		}
		items = append(items, i)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return 0, e
	}
	n := 0
	for _, i := range items {
		if e = store.Delete(ctx, i.object); e != nil {
			return n, e
		}
		if _, e = p.Exec(ctx, `DELETE FROM upload_intents WHERE id=$1 AND state<>'CONSUMED' AND expires_at<now()-interval '1 hour'`, i.id); e != nil {
			return n, e
		}
		n++
	}
	return n, nil
}
