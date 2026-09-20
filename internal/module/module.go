package module

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	SemesterID   *string `json:"semester_id"`
	ModuleCode   *string `json:"module_code"`
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	LecturerName *string `json:"lecturer_name"`
}

func (in Input) Validate(create bool) error {
	if create && in.SemesterID == nil {
		return apperror.ErrInvalid
	}
	if in.SemesterID != nil {
		if len(*in.SemesterID) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.SemesterID) == "" {
			return apperror.ErrInvalid
		}
		if _, e := uuid.Parse(*in.SemesterID); e != nil {
			return apperror.ErrInvalid
		}
	}
	if create && in.ModuleCode == nil {
		return apperror.ErrInvalid
	}
	if in.ModuleCode != nil {
		if len(*in.ModuleCode) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.ModuleCode) == "" {
			return apperror.ErrInvalid
		}
	}
	if create && in.Name == nil {
		return apperror.ErrInvalid
	}
	if in.Name != nil {
		if len(*in.Name) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.Name) == "" {
			return apperror.ErrInvalid
		}
	}
	if in.Description != nil {
		if len(*in.Description) > 20000 {
			return apperror.ErrInvalid
		}
	}
	if in.LecturerName != nil {
		if len(*in.LecturerName) > 20000 {
			return apperror.ErrInvalid
		}
	}
	return nil
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "module.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "module.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,batch_id,semester_id,module_code,name,description,lecturer_name,status,created_at,updated_at FROM modules WHERE batch_id=$1  AND status<>'ARCHIVED' AND ($3 OR NOT $3) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $4)t`, batch, limit, manage, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "module.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "module.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT id,batch_id,semester_id,module_code,name,description,lecturer_name,status,created_at,updated_at FROM modules WHERE batch_id=$1 AND id=$2  AND status<>'ARCHIVED' AND ($3 OR NOT $3))t`, batch, id, manage))
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "module.manage"); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `INSERT INTO modules(batch_id,semester_id,module_code,name,description,lecturer_name) VALUES($1,COALESCE($2::uuid,NULL),COALESCE($3,''),COALESCE($4,''),COALESCE($5,''),COALESCE($6,'')) RETURNING id`, batch, in.SemesterID, in.ModuleCode, in.Name, in.Description, in.LecturerName).Scan(&id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "MODULE_CREATED", "module", id, nil)
	})
	return id, e
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.Validate(false); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "module.manage"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE modules SET semester_id=COALESCE($3,semester_id),module_code=COALESCE($4,module_code),name=COALESCE($5,name),description=COALESCE($6,description),lecturer_name=COALESCE($7,lecturer_name),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, in.SemesterID, in.ModuleCode, in.Name, in.Description, in.LecturerName)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "MODULE_UPDATED", "module", id, nil)
	})
}
func (s Service) Transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		permission := "module.manage"
		target := "ARCHIVED"
		if publish {
			permission = "module.manage"
			target = "PUBLISHED"
		}
		if e := authorization.Write(ctx, tx, user, batch, permission); e != nil {
			return e
		}
		var old string
		if e := tx.QueryRow(ctx, `SELECT status FROM modules WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&old); e != nil {
			return e
		}
		if old == target {
			return nil
		}
		if publish && old != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE modules SET status=$3,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, target); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "MODULE_"+target, "module", id, nil)
	})
}
