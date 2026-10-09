package audit

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
)

func Record(ctx context.Context, tx pgx.Tx, batch, actor, action, kind, id string, values any) error {
	b, e := json.Marshal(values)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_logs(batch_id,actor_user_id,action,entity_type,entity_id,new_values) VALUES(NULLIF($1::text,'')::uuid,NULLIF($2::text,'')::uuid,$3,$4,$5::uuid,$6)`, batch, actor, action, kind, id, b)
	return e
}
