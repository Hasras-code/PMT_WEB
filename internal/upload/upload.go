package upload

import (
	"context"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type Store interface {
	Sign(storage.Grant) (string, error)
	Verify(string, string) (storage.Grant, error)
	Put(context.Context, string, string, int64, io.Reader) (storage.Metadata, error)
	Inspect(string, string) (storage.Metadata, error)
	DownloadURL(string, string, string, string) (string, error)
}
type Service struct {
	Pool    *pgxpool.Pool
	Store   Store
	BaseURL string
}
type Input struct {
	FileName string `json:"file_name"`
	MIME     string `json:"mime_type"`
	Size     int64  `json:"size_bytes"`
}
type Object struct {
	ID, Key, Name, MIME string
	Size                int64
}

func (s Service) Authorize(ctx context.Context, user, batch, purpose, permission string, in Input) (json.RawMessage, error) {
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
	id, key := uuid.NewString(), storage.Key()
	expires := time.Now().Add(15 * time.Minute)
	_, e := s.Pool.Exec(ctx, `INSERT INTO upload_intents(id,owner_id,batch_id,purpose,storage_key,mime_type,size_bytes,file_name,expires_at) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9)`, id, user, batch, purpose, key, in.MIME, in.Size, in.FileName, expires)
	if e != nil {
		return nil, e
	}
	token, e := s.Store.Sign(storage.Grant{Key: key, ID: id, Purpose: "upload", Expiry: time.Now().Add(5 * time.Minute).Unix()})
	if e != nil {
		return nil, e
	}
	b, e := json.Marshal(map[string]any{"upload_id": id, "upload_url": s.BaseURL + "/v1/files/uploads/" + token, "method": "PUT", "headers": map[string]string{"Content-Type": in.MIME}, "expires_in": 300})
	return b, e
}
func (s Service) Receive(ctx context.Context, token, mime string, length int64, r io.Reader) error {
	g, e := s.Store.Verify(token, "upload")
	if e != nil {
		return e
	}
	var obj Object
	var state, owner string
	var batch *string
	e = s.Pool.QueryRow(ctx, `SELECT storage_key,mime_type,size_bytes,state,owner_id,batch_id FROM upload_intents WHERE id=$1 AND expires_at>now()`, g.ID).Scan(&obj.Key, &obj.MIME, &obj.Size, &state, &owner, &batch)
	if e != nil {
		return db.Error(e)
	}
	if state != "PENDING" || g.Key != obj.Key {
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
	if _, e = s.Store.Put(ctx, obj.Key, obj.MIME, obj.Size, r); e != nil {
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
func Consume(ctx context.Context, tx pgx.Tx, user, batch, id, purpose string) (Object, error) {
	var o Object
	o.ID = id
	e := tx.QueryRow(ctx, `UPDATE upload_intents SET state='CONSUMED' WHERE id=$1 AND owner_id=$2 AND batch_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid AND purpose=$4 AND state='UPLOADED' AND expires_at>now() RETURNING storage_key,file_name,mime_type,size_bytes`, id, user, batch, purpose).Scan(&o.Key, &o.Name, &o.MIME, &o.Size)
	return o, db.Error(e)
}
