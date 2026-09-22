package resource

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Service struct{ Pool *pgxpool.Pool }
type Input struct {
	ModuleID           string  `json:"module_id"`
	UploadID           string  `json:"upload_id"`
	Type               string  `json:"type"`
	Title              string  `json:"title"`
	Description        string  `json:"description"`
	AcademicYear       *string `json:"academic_year"`
	ExamType           *string `json:"exam_type"`
	CompressionProfile string  `json:"compression_profile"`
	OriginalSize       int64   `json:"original_size_bytes"`
	PageCount          *int    `json:"page_count"`
	ChangeNote         string  `json:"change_note"`
}
type Update struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}
type Filter struct {
	ModuleID string
	Type     string
	Query    string
}

func validType(value string) bool {
	switch value {
	case "LECTURE_NOTE", "HANDWRITTEN_NOTE", "PAST_PAPER", "TUTORIAL", "ASSIGNMENT", "REFERENCE", "OTHER":
		return true
	default:
		return false
	}
}

func (filter *Filter) validate() error {
	filter.Query = strings.TrimSpace(filter.Query)
	if filter.ModuleID != "" {
		if _, e := uuid.Parse(filter.ModuleID); e != nil {
			return apperror.ErrInvalid
		}
	}
	if filter.Type != "" && !validType(filter.Type) {
		return apperror.ErrInvalid
	}
	if len(filter.Query) > 200 {
		return apperror.ErrInvalid
	}
	return nil
}

func (in Input) validate() error {
	if _, e := uuid.Parse(in.UploadID); e != nil {
		return apperror.ErrInvalid
	}
	if in.PageCount != nil && *in.PageCount < 1 {
		return apperror.ErrInvalid
	}
	if len(in.ChangeNote) > 2000 {
		return apperror.ErrInvalid
	}
	switch in.CompressionProfile {
	case "", "NONE", "LOW_SIZE", "BALANCED", "HIGH_QUALITY":
	default:
		return apperror.ErrInvalid
	}
	return nil
}
func (s Service) Create(ctx context.Context, user, batch string, in Input) (string, error) {
	if e := in.validate(); e != nil {
		return "", e
	}
	if _, e := uuid.Parse(in.ModuleID); e != nil {
		return "", apperror.ErrInvalid
	}
	if in.Title == "" || len(in.Title) > 300 || len(in.Description) > 20000 {
		return "", apperror.ErrInvalid
	}
	if !validType(in.Type) {
		return "", apperror.ErrInvalid
	}
	id := uuid.NewString()
	e := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.WriteWithPlatform(ctx, tx, user, batch, "resource.create", "platform_user.manage"); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `INSERT INTO resources(id,batch_id,module_id,uploaded_by,type,title,description,academic_year,exam_type) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, batch, in.ModuleID, user, in.Type, in.Title, in.Description, in.AcademicYear, in.ExamType); e != nil {
			return e
		}
		if e := addVersion(ctx, tx, user, batch, id, in); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "RESOURCE_CREATED", "resource", id, nil)
	})
	return id, e
}
func addVersion(ctx context.Context, tx pgx.Tx, user, batch, id string, in Input) error {
	o, e := upload.Consume(ctx, tx, user, batch, in.UploadID, "resource")
	if e != nil {
		return e
	}
	if o.MIME != "application/pdf" {
		return apperror.ErrInvalid
	}
	if in.OriginalSize == 0 {
		in.OriginalSize = o.Size
	}
	if in.OriginalSize < o.Size {
		return apperror.ErrInvalid
	}
	if in.CompressionProfile == "" {
		in.CompressionProfile = "NONE"
	}
	var version string
	e = tx.QueryRow(ctx, `INSERT INTO resource_versions(batch_id,resource_id,version_number,storage_key,file_name,mime_type,size_bytes,original_size_bytes,compression_profile,page_count,uploaded_by,change_note) SELECT $1,$2,COALESCE(max(version_number),0)+1,$3,$4,$5,$6,$7,$8,$9,$10,$11 FROM resource_versions WHERE resource_id=$2 RETURNING id`, batch, id, o.Key, o.Name, o.MIME, o.Size, in.OriginalSize, in.CompressionProfile, in.PageCount, user, in.ChangeNote).Scan(&version)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `UPDATE resources SET current_version_id=$3,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, version)
	return e
}
func (s Service) AddVersion(ctx context.Context, user, batch, id string, in Input) error {
	if e := in.validate(); e != nil {
		return e
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.WriteWithPlatform(ctx, tx, user, batch, "resource.update", "platform_user.manage"); e != nil {
			return e
		}
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM resources WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&status); e != nil {
			return e
		}
		if status == "ARCHIVED" {
			return apperror.ErrConflict
		}
		if status == "PUBLISHED" {
			if e := authorization.RequireWithPlatform(ctx, tx, user, batch, "resource.publish", "platform_user.manage"); e != nil {
				return e
			}
		}
		if e := addVersion(ctx, tx, user, batch, id, in); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "RESOURCE_VERSION_CREATED", "resource", id, nil)
	})
}

const projection = `r.id,r.batch_id,r.module_id,m.module_code,m.name AS module_name,r.type,r.title,r.description,r.academic_year,r.exam_type,r.status,r.published_at,r.created_at,r.updated_at,v.id AS version_id,v.version_number,v.file_name,v.mime_type,v.size_bytes,v.original_size_bytes,v.compression_profile,v.page_count`

func (s Service) List(ctx context.Context, user, batch string, filter Filter, limit, offset int) (json.RawMessage, error) {
	if e := filter.validate(); e != nil {
		return nil, e
	}
	if e := authorization.RequireWithPlatform(ctx, s.Pool, user, batch, "resource.view", "platform_user.manage"); e != nil {
		return nil, e
	}
	manage, e := authorization.CanWithPlatform(ctx, s.Pool, user, batch, "resource.update", "platform_user.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT `+projection+` FROM resources r JOIN modules m ON m.id=r.module_id AND m.batch_id=r.batch_id JOIN resource_versions v ON v.id=r.current_version_id WHERE r.batch_id=$1 AND (r.status='PUBLISHED' OR ($2 AND r.status='DRAFT')) AND ($3='' OR r.module_id=NULLIF($3,'')::uuid) AND ($4='' OR r.type=$4) AND ($5='' OR r.title ILIKE '%'||$5||'%' OR r.description ILIKE '%'||$5||'%' OR m.module_code ILIKE '%'||$5||'%' OR m.name ILIKE '%'||$5||'%') ORDER BY m.module_code,r.created_at DESC,r.id DESC LIMIT $6 OFFSET $7)t`, batch, manage, filter.ModuleID, filter.Type, filter.Query, limit, offset))
}
func (s Service) Get(ctx context.Context, user, batch, id string) (json.RawMessage, error) {
	if e := authorization.RequireWithPlatform(ctx, s.Pool, user, batch, "resource.view", "platform_user.manage"); e != nil {
		return nil, e
	}
	manage, e := authorization.CanWithPlatform(ctx, s.Pool, user, batch, "resource.update", "platform_user.manage")
	if e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT row_to_json(t) FROM (SELECT `+projection+` FROM resources r JOIN modules m ON m.id=r.module_id AND m.batch_id=r.batch_id JOIN resource_versions v ON v.id=r.current_version_id WHERE r.batch_id=$1 AND r.id=$2 AND (r.status='PUBLISHED' OR ($3 AND r.status='DRAFT')))t`, batch, id, manage))
}
func (s Service) Versions(ctx context.Context, user, batch, id string, limit, offset int) (json.RawMessage, error) {
	if _, e := s.Get(ctx, user, batch, id); e != nil {
		return nil, e
	}
	return db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM (SELECT id,version_number,file_name,mime_type,size_bytes,original_size_bytes,compression_profile,page_count,change_note,created_at FROM resource_versions WHERE batch_id=$1 AND resource_id=$2 ORDER BY version_number DESC LIMIT $3 OFFSET $4)t`, batch, id, limit, offset))
}
func (s Service) Download(ctx context.Context, user, batch, id, version string) (upload.Object, error) {
	var o upload.Object
	if _, e := s.Get(ctx, user, batch, id); e != nil {
		return o, e
	}
	e := s.Pool.QueryRow(ctx, `SELECT v.storage_key,v.file_name,v.mime_type,v.size_bytes FROM resource_versions v JOIN resources r ON r.id=v.resource_id AND r.batch_id=v.batch_id WHERE r.batch_id=$1 AND r.id=$2 AND v.id=COALESCE(NULLIF($3,'')::uuid,r.current_version_id)`, batch, id, version).Scan(&o.Key, &o.Name, &o.MIME, &o.Size)
	return o, db.Error(e)
}
func (s Service) Update(ctx context.Context, user, batch, id string, in Update) error {
	if in.Title != nil && (*in.Title == "" || len(*in.Title) > 300) {
		return apperror.ErrInvalid
	}
	if in.Description != nil && len(*in.Description) > 20000 {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.WriteWithPlatform(ctx, tx, user, batch, "resource.update", "platform_user.manage"); e != nil {
			return e
		}
		var state string
		if e := tx.QueryRow(ctx, `SELECT status FROM resources WHERE batch_id=$1 AND id=$2`, batch, id).Scan(&state); e != nil {
			return e
		}
		if state == "PUBLISHED" {
			if e := authorization.RequireWithPlatform(ctx, tx, user, batch, "resource.publish", "platform_user.manage"); e != nil {
				return e
			}
		}
		tag, e := tx.Exec(ctx, `UPDATE resources SET title=COALESCE($3,title),description=COALESCE($4,description),updated_at=now() WHERE batch_id=$1 AND id=$2 AND status<>'ARCHIVED'`, batch, id, in.Title, in.Description)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return nil
	})
}
func (s Service) Transition(ctx context.Context, user, batch, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		perm, target := "resource.delete", "ARCHIVED"
		if publish {
			perm, target = "resource.publish", "PUBLISHED"
		}
		if e := authorization.WriteWithPlatform(ctx, tx, user, batch, perm, "platform_user.manage"); e != nil {
			return e
		}
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM resources WHERE batch_id=$1 AND id=$2 FOR UPDATE`, batch, id).Scan(&status); e != nil {
			return e
		}
		if status == target {
			return nil
		}
		if publish && status != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE resources SET status=$3,published_at=CASE WHEN $3='PUBLISHED' THEN now() ELSE published_at END,updated_at=now() WHERE batch_id=$1 AND id=$2`, batch, id, target); e != nil {
			return e
		}
		return audit.Record(ctx, tx, batch, user, "RESOURCE_"+target, "resource", id, nil)
	})
}
func (s Service) Bookmark(ctx context.Context, user, batch, id string, remove bool) error {
	if _, e := s.Get(ctx, user, batch, id); e != nil {
		return e
	}
	if remove {
		_, e := s.Pool.Exec(ctx, `DELETE FROM bookmarks WHERE user_id=$1 AND batch_id=$2 AND resource_id=$3`, user, batch, id)
		return e
	}
	_, e := s.Pool.Exec(ctx, `INSERT INTO bookmarks(user_id,batch_id,resource_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, user, batch, id)
	return e
}
