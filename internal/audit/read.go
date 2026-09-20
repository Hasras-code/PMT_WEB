package audit

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func List(ctx context.Context, p *pgxpool.Pool, user, batch string, limit, offset int) (json.RawMessage, error) {
	if batch == "" {
		if e := authorization.RequirePlatform(ctx, p, user, "platform_audit.view"); e != nil {
			return nil, e
		}
	} else if e := authorization.Require(ctx, p, user, batch, "audit.view"); e != nil {
		return nil, e
	}
	return db.JSON(p.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,batch_id,actor_user_id,action,entity_type,entity_id,new_values,created_at FROM audit_logs WHERE ($1='' OR batch_id=NULLIF($1,'')::uuid) AND ($1='' OR entity_type<>'complaint') ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3)t`, batch, limit, offset))
}
