package audit

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Record(ctx context.Context, tx pgx.Tx, batch, actor, action, kind, id string, values any) error {
	batchID, e := uuidParam(batch, true)
	if e != nil {
		return errors.New("audit batch ID is invalid")
	}
	actorID, e := uuidParam(actor, true)
	if e != nil {
		return errors.New("audit actor ID is invalid")
	}
	entityID, e := uuidParam(id, false)
	if e != nil {
		return errors.New("audit entity ID is invalid")
	}
	b, e := json.Marshal(values)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_logs(batch_id,actor_user_id,action,entity_type,entity_id,new_values) VALUES($1,$2,$3,$4,$5,$6::jsonb)`, batchID, actorID, action, kind, entityID, string(b))
	return e
}

func uuidParam(value string, allowEmpty bool) (pgtype.UUID, error) {
	if value == "" && allowEmpty {
		return pgtype.UUID{}, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: [16]byte(parsed), Valid: true}, nil
}
