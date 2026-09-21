package batch

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	EntryYear      int    `json:"entry_year"`
	GraduationYear *int   `json:"graduation_year"`
	Description    string `json:"description"`
}
type Profile struct {
	Headline *string `json:"headline"`
	About    *string `json:"about_text"`
	Mission  *string `json:"mission_text"`
	Contact  *string `json:"contact_email"`
	UploadID string  `json:"upload_id,omitempty"`
}

func (s Service) Create(ctx context.Context, user string, in Input, operator bool) (string, error) {
	if in.Name == "" || len(in.Name) > 200 || !regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`).MatchString(in.Slug) || len(in.Slug) > 100 || in.EntryYear < 1900 || in.EntryYear > 2200 || len(in.Description) > 20000 {
		return "", apperror.ErrInvalid
	}
	var id string
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if !operator {
			if e := authorization.RequirePlatform(ctx, tx, user, "batch.create"); e != nil {
				return e
			}
		}
		if e := tx.QueryRow(ctx, `INSERT INTO batches(name,slug,entry_year,graduation_year,description) VALUES($1,$2,$3,$4,$5) RETURNING id`, in.Name, in.Slug, in.EntryYear, in.GraduationYear, in.Description).Scan(&id); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `INSERT INTO batch_profiles(batch_id) VALUES($1)`, id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, id, user, "BATCH_CREATED", "batch", id, nil)
	})
	return id, e
}
func (s Service) List(ctx context.Context, user string, limit, offset int) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT b.id,b.name,b.slug,b.entry_year,b.graduation_year,b.description,b.status FROM batches b JOIN batch_memberships m ON m.batch_id=b.id WHERE m.user_id=$1 AND m.status='ACTIVE' AND b.status<>'ARCHIVED' ORDER BY b.entry_year DESC,b.id DESC LIMIT $2 OFFSET $3)t`, user, limit, offset))
}
func (s Service) Get(ctx context.Context, user, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, id, "batch.view"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM(SELECT id,name,slug,entry_year,graduation_year,description,status FROM batches WHERE id=$1)t`, id))
}

func (s Service) Summary(ctx context.Context, user, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, id, "batch.view"); e != nil {
		return nil, e
	}
	resourceManage, e := authorization.Can(ctx, s.Pool, user, id, "resource.update")
	if e != nil {
		return nil, e
	}
	announcementManage, e := authorization.Can(ctx, s.Pool, user, id, "announcement.update")
	if e != nil {
		return nil, e
	}
	eventManage, e := authorization.Can(ctx, s.Pool, user, id, "event.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT json_build_object(
		'members',(SELECT count(*) FROM batch_memberships WHERE batch_id=$1 AND status='ACTIVE'),
		'resources',(SELECT count(*) FROM resources WHERE batch_id=$1 AND status<>'ARCHIVED' AND (status='PUBLISHED' OR $2)),
		'announcements',(SELECT count(*) FROM announcements WHERE batch_id=$1 AND status<>'ARCHIVED' AND (status='PUBLISHED' OR $3) AND (expires_at IS NULL OR expires_at>now() OR $3)),
		'events',(SELECT count(*) FROM events WHERE batch_id=$1 AND status<>'ARCHIVED' AND (status='PUBLISHED' OR $4))
	)`, id, resourceManage, announcementManage, eventManage))
}
func (s Service) Update(ctx context.Context, user, id string, name, description *string) error {
	if name != nil && (*name == "" || len(*name) > 200) {
		return apperror.ErrInvalid
	}
	if description != nil && len(*description) > 20000 {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, id, "batch.manage"); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE batches SET name=COALESCE($2,name),description=COALESCE($3,description),updated_at=now() WHERE id=$1`, id, name, description)
		return e
	})
}
func (s Service) Archive(ctx context.Context, user, id string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.RequirePlatform(ctx, tx, user, "batch.archive"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE batches SET status='ARCHIVED',updated_at=now() WHERE id=$1 AND status<>'ARCHIVED'`, id)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		return audit.Record(ctx, tx, id, user, "BATCH_ARCHIVED", "batch", id, nil)
	})
}
func (s Service) Profile(ctx context.Context, user, id string) (json.RawMessage, error) {
	if e := authorization.Require(ctx, s.Pool, user, id, "batch.view"); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM(SELECT batch_id,headline,about_text,mission_text,contact_email,updated_at FROM batch_profiles WHERE batch_id=$1)t`, id))
}
func (s Service) SetProfile(ctx context.Context, user, id string, in Profile) error {
	if (in.Headline != nil && len(*in.Headline) > 300) || (in.About != nil && len(*in.About) > 20000) || (in.Mission != nil && len(*in.Mission) > 20000) || (in.Contact != nil && len(*in.Contact) > 254) {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.Write(ctx, tx, user, id, "batch.profile.manage"); e != nil {
			return e
		}
		var key *string
		if in.UploadID != "" {
			o, e := upload.Consume(ctx, tx, user, id, in.UploadID, "batch")
			if e != nil {
				return e
			}
			key = &o.Key
		}
		_, e := tx.Exec(ctx, `INSERT INTO batch_profiles(batch_id,headline,about_text,mission_text,contact_email,hero_image_key,updated_by) VALUES($1,COALESCE($2,''),COALESCE($3,''),COALESCE($4,''),COALESCE($5,''),$6,$7) ON CONFLICT(batch_id) DO UPDATE SET headline=COALESCE($2,batch_profiles.headline),about_text=COALESCE($3,batch_profiles.about_text),mission_text=COALESCE($4,batch_profiles.mission_text),contact_email=COALESCE($5,batch_profiles.contact_email),hero_image_key=COALESCE($6,batch_profiles.hero_image_key),updated_by=$7,updated_at=now()`, id, in.Headline, in.About, in.Mission, in.Contact, key, user)
		return e
	})
}
func (s Service) Public(ctx context.Context, slug string) (json.RawMessage, error) {
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM(SELECT b.id,b.name,b.slug,b.entry_year,b.graduation_year,b.description,p.headline,p.about_text,p.mission_text,p.contact_email,CASE WHEN p.hero_image_key IS NOT NULL THEN '/v1/public/batches/'||b.slug||'/image' END AS hero_image_url FROM batches b LEFT JOIN batch_profiles p ON p.batch_id=b.id WHERE b.slug=$1 AND b.status<>'ARCHIVED')t`, slug))
}
func (s Service) PublicEvents(ctx context.Context, slug, id string, limit, offset int) (json.RawMessage, error) {
	b, e := db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT e.id,e.title,e.description,e.location,e.starts_at,e.ends_at,CASE WHEN e.cover_image_key IS NOT NULL THEN '/v1/public/batches/'||b.slug||'/events/'||e.id||'/image' END AS cover_image_url FROM events e JOIN batches b ON b.id=e.batch_id WHERE b.slug=$1 AND b.status<>'ARCHIVED' AND e.status='PUBLISHED' AND e.visibility='PUBLIC' AND ($2='' OR e.id=NULLIF($2,'')::uuid) ORDER BY e.starts_at DESC,e.id DESC LIMIT $3 OFFSET $4)t`, slug, id, limit, offset))
	if id != "" {
		return db.One(b, e)
	}
	return b, e
}
