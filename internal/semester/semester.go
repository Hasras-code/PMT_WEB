package semester

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	SemesterNumber *int       `json:"semester_number"`
	Name           *string    `json:"name"`
	AcademicYear   *string    `json:"academic_year"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
}

func (in Input) Validate(create bool) error {
	if create && in.SemesterNumber == nil {
		return apperror.ErrInvalid
	}
	if in.SemesterNumber != nil && (*in.SemesterNumber < 1 || *in.SemesterNumber > 100000) {
		return apperror.ErrInvalid
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
	if create && in.AcademicYear == nil {
		return apperror.ErrInvalid
	}
	if in.AcademicYear != nil {
		if len(*in.AcademicYear) > 20000 {
			return apperror.ErrInvalid
		}
		if strings.TrimSpace(*in.AcademicYear) == "" {
			return apperror.ErrInvalid
		}
	}
	return nil
}
func (s Service) List(ctx context.Context, user, batch string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "semester.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "semester.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,batch_id,semester_number,name,academic_year,starts_at,ends_at,is_current,created_at,updated_at FROM semesters WHERE batch_id=$1  AND ($3 OR NOT $3) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $4)t`, batch, limit, manage, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, batch, "semester.view"); e != nil {
		return nil, e
	}
	manage, e := authorization.Can(ctx, s.Pool, user, batch, "semester.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT id,batch_id,semester_number,name,academic_year,starts_at,ends_at,is_current,created_at,updated_at FROM semesters WHERE batch_id=$1 AND id=$2  AND ($3 OR NOT $3))t`, batch, id, manage))
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.Validate(true); e != nil {
		return "", e
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "semester.manage"); e != nil {
			return e
		}
		if e := tx.QueryRow(ctx, `INSERT INTO semesters(batch_id,semester_number,name,academic_year,starts_at,ends_at) VALUES($1,COALESCE($2,0),COALESCE($3,''),COALESCE($4,''),$5::timestamptz,$6::timestamptz) RETURNING id`, batch, in.SemesterNumber, in.Name, in.AcademicYear, in.StartsAt, in.EndsAt).Scan(&id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "SEMESTER_CREATED", "semester", id, nil)
	})
	return id, e
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.Validate(false); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "semester.manage"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE semesters SET semester_number=COALESCE($3,semester_number),name=COALESCE($4,name),academic_year=COALESCE($5,academic_year),starts_at=COALESCE($6,starts_at),ends_at=COALESCE($7,ends_at),updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, in.SemesterNumber, in.Name, in.AcademicYear, in.StartsAt, in.EndsAt)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, batch, user, "SEMESTER_UPDATED", "semester", id, nil)
	})
}
func (s Service) SetCurrent(ctx context.Context, user, batch, id string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, batch, "semester.manage"); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `UPDATE semesters SET is_current=false WHERE batch_id=$1`, batch); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE semesters SET is_current=true,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return nil
	})
}
