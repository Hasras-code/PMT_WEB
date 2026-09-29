package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Pool    *pgxpool.Pool
	Store   *storage.Manager
	BaseURL string
}
type Input struct {
	FileName string `json:"file_name"`
	MIME     string `json:"mime_type"`
	Size     int64  `json:"size_bytes"`
}
type Object struct {
	ID, Key, Name, MIME string
	Provider, Class     string
	Size                int64
}

func (o Object) StoredObject() storage.Object {
	return storage.Object{Provider: o.Provider, Class: o.Class, Key: o.Key, Name: o.Name, MIME: o.MIME, Size: o.Size}
}

// AuthorizeChecked creates an upload intent after a feature service has
// performed its resource-specific authorization. It is used when a generic
// batch permission is insufficient, such as an assignment to one exact fund.
func (s Service) AuthorizeChecked(ctx context.Context, user, batch, purpose, prefix string, in Input) (json.RawMessage, error) {
	if in.Size < 1 || in.Size > 50<<20 || in.FileName == "" || len(in.FileName) > 255 || strings.ContainsAny(in.FileName, "/\\\r\n\x00") || in.MIME != "application/pdf" || strings.ToLower(filepath.Ext(in.FileName)) != ".pdf" {
		return nil, apperror.ErrInvalid
	}
	id := uuid.NewString()
	object, err := s.Store.NewObject(prefix, in.FileName, in.MIME, in.Size, storage.ClassPrivate)
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(15 * time.Minute)
	if _, err := s.Pool.Exec(ctx, `INSERT INTO upload_intents(id,owner_id,batch_id,purpose,storage_key,mime_type,size_bytes,file_name,expires_at,storage_provider,storage_class) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, user, batch, purpose, object.Key, in.MIME, in.Size, in.FileName, expires, object.Provider, object.Class); err != nil {
		return nil, err
	}
	upload, err := s.Store.CreateUpload(ctx, object, id)
	if err != nil {
		return nil, err
	}
	return uploadResponse(id, s.BaseURL, upload)
}

func (s Service) Authorize(ctx context.Context, user, batch, purpose, permission, entity string, in Input) (json.RawMessage, error) {
	if in.Size < 1 || len(in.FileName) > 255 || in.FileName == "" || strings.ContainsAny(in.FileName, "/\\\r\n\x00") {
		return nil, apperror.ErrInvalid
	}
	image := purpose != "resource" && purpose != "attachment"
	ext := strings.ToLower(filepath.Ext(in.FileName))
	if image {
		if in.Size > 10<<20 {
			return nil, apperror.ErrInvalid
		}
		valid := map[string][]string{"image/jpeg": {".jpg", ".jpeg"}, "image/png": {".png"}, "image/webp": {".webp"}}
		ok := false
		for _, x := range valid[in.MIME] {
			ok = ok || x == ext
		}
		if !ok {
			return nil, apperror.ErrInvalid
		}
	} else if in.MIME != "application/pdf" || ext != ".pdf" || in.Size > 50<<20 {
		return nil, apperror.ErrInvalid
	}
	if batch != "" {
		var e error
		if purpose == "resource" {
			e = authorization.RequireWithPlatform(ctx, s.Pool, user, batch, permission, "platform_user.manage")
		} else {
			e = authorization.Require(ctx, s.Pool, user, batch, permission)
		}
		if e != nil {
			return nil, e
		}
	} else if purpose == "gallery" {
		if e := authorization.RequirePlatform(ctx, s.Pool, user, "gallery.manage"); e != nil {
			return nil, e
		}
	} else if purpose != "profile" {
		return nil, apperror.ErrInvalid
	}
	class := storage.ClassPrivate
	if purpose == "gallery" || purpose == "batch" {
		class = storage.ClassPublic
	}
	id := uuid.NewString()
	prefix := objectPrefix(user, batch, purpose, entity)
	object, err := s.Store.NewObject(prefix, in.FileName, in.MIME, in.Size, class)
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(15 * time.Minute)
	_, e := s.Pool.Exec(ctx, `INSERT INTO upload_intents(id,owner_id,batch_id,purpose,storage_key,mime_type,size_bytes,file_name,expires_at,storage_provider,storage_class) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11)`, id, user, batch, purpose, object.Key, in.MIME, in.Size, in.FileName, expires, object.Provider, object.Class)
	if e != nil {
		return nil, e
	}
	upload, e := s.Store.CreateUpload(ctx, object, id)
	if e != nil {
		return nil, e
	}
	return uploadResponse(id, s.BaseURL, upload)
}
func (s Service) Receive(ctx context.Context, token, mime string, length int64, r io.Reader) error {
	g, e := s.Store.VerifyLocal(token, "upload")
	if e != nil {
		return e
	}
	var obj Object
	var state, owner string
	var batch *string
	e = s.Pool.QueryRow(ctx, `SELECT storage_key,mime_type,size_bytes,state,owner_id,batch_id,storage_provider,storage_class FROM upload_intents WHERE id=$1 AND expires_at>now()`, g.ID).Scan(&obj.Key, &obj.MIME, &obj.Size, &state, &owner, &batch, &obj.Provider, &obj.Class)
	if e != nil {
		return db.Error(e)
	}
	if state != "PENDING" || g.Key != obj.Key || obj.Provider != storage.ProviderLocal {
		return apperror.ErrConflict
	}
	if mime != obj.MIME || (length >= 0 && length != obj.Size) {
		return apperror.ErrInvalid
	}
	var active bool
	if e = s.Pool.QueryRow(ctx, `SELECT status='ACTIVE' FROM users WHERE id=$1`, owner).Scan(&active); e != nil {
		return e
	}
	if !active {
		return apperror.ErrForbidden
	}
	if _, e = s.Store.Put(ctx, obj.StoredObject(), r); e != nil {
		return e
	}
	tag, e := s.Pool.Exec(ctx, `UPDATE upload_intents SET state='UPLOADED' WHERE id=$1 AND state='PENDING' AND expires_at>now()`, g.ID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return apperror.ErrConflict
	}
	return nil
}

func (s Service) Confirm(ctx context.Context, user, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return apperror.ErrInvalid
	}
	var object Object
	var state string
	err := s.Pool.QueryRow(ctx, `SELECT storage_key,file_name,mime_type,size_bytes,state,storage_provider,storage_class FROM upload_intents WHERE id=$1 AND owner_id=$2 AND expires_at>now()`, id, user).Scan(&object.Key, &object.Name, &object.MIME, &object.Size, &state, &object.Provider, &object.Class)
	if err != nil {
		return db.Error(err)
	}
	if state == "UPLOADED" {
		return nil
	}
	if state != "PENDING" {
		return apperror.ErrConflict
	}
	metadata, err := s.Store.Confirm(ctx, object.StoredObject())
	if err != nil {
		return err
	}
	if metadata.Size != object.Size || metadata.MIME != object.MIME {
		return apperror.ErrInvalid
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE upload_intents SET state='UPLOADED' WHERE id=$1 AND owner_id=$2 AND state='PENDING' AND expires_at>now()`, id, user)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return apperror.ErrConflict
	}
	return nil
}

func Consume(ctx context.Context, tx pgx.Tx, user, batch, id, purpose string) (Object, error) {
	var o Object
	o.ID = id
	e := tx.QueryRow(ctx, `UPDATE upload_intents SET state='CONSUMED' WHERE id=$1 AND owner_id=$2 AND batch_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid AND purpose=$4 AND state='UPLOADED' AND expires_at>now() RETURNING storage_key,file_name,mime_type,size_bytes,storage_provider,storage_class`, id, user, batch, purpose).Scan(&o.Key, &o.Name, &o.MIME, &o.Size, &o.Provider, &o.Class)
	return o, db.Error(e)
}

func uploadResponse(id, baseURL string, upload storage.UploadAuthorization) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"upload_id":   id,
		"upload_url":  upload.URL,
		"confirm_url": baseURL + "/v1/files/uploads/" + id + "/confirm",
		"method":      "PUT",
		"headers":     upload.Headers,
		"expires_in":  upload.ExpiresIn,
	})
}

func objectPrefix(user, batch, purpose, entity string) string {
	switch purpose {
	case "profile":
		return path.Join("users", user, "profile")
	case "gallery":
		return "gallery/uploads"
	case "batch":
		return path.Join("batches", batch, "profile")
	case "resource":
		if entity != "" {
			return path.Join("batches", batch, "resources", entity, "versions")
		}
		return path.Join("batches", batch, "resources", "uploads")
	case "attachment":
		return path.Join("batches", batch, "announcements", entity, "attachments")
	case "event":
		return path.Join("batches", batch, "events", entity, "cover")
	default:
		return path.Join("uploads", fmt.Sprintf("%s-%s", purpose, entity))
	}
}
