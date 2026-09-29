package gallery

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/authorization"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type Service struct {
	Pool    *pgxpool.Pool
	Store   *storage.Manager
	BaseURL string
	Secret  []byte
}
type Input struct {
	Title             string     `json:"title"`
	Caption           string     `json:"caption"`
	AltText           string     `json:"alt_text"`
	DisplayUploadID   string     `json:"display_upload_id"`
	ThumbnailUploadID string     `json:"thumbnail_upload_id"`
	TakenAt           *time.Time `json:"taken_at"`
}
type Update struct {
	Title   *string `json:"title"`
	Caption *string `json:"caption"`
	AltText *string `json:"alt_text"`
}
type Image struct {
	ID                                 string     `json:"id"`
	Title                              string     `json:"title"`
	Caption                            string     `json:"caption"`
	AltText                            string     `json:"alt_text"`
	Width                              int        `json:"width"`
	Height                             int        `json:"height"`
	TakenAt                            *time.Time `json:"taken_at"`
	PublishedAt                        *time.Time `json:"published_at"`
	DisplayURL                         string     `json:"display_url"`
	ThumbnailURL                       string     `json:"thumbnail_url"`
	Status                             string     `json:"status,omitempty"`
	displayKey, thumbnailKey           string
	displayProvider, thumbnailProvider string
	displayClass, thumbnailClass       string
}
type Page struct {
	Data       []Image `json:"data"`
	NextCursor string  `json:"next_cursor,omitempty"`
}
type cursor struct {
	At    time.Time `json:"at"`
	ID    string    `json:"id"`
	Month string    `json:"month"`
}

func (s Service) Create(ctx context.Context, user string, in Input) (string, error) {
	if strings.TrimSpace(in.AltText) == "" || len(in.AltText) > 1000 || len(in.Title) > 300 || len(in.Caption) > 5000 || in.DisplayUploadID == in.ThumbnailUploadID {
		return "", apperror.ErrInvalid
	}
	if _, e := uuid.Parse(in.DisplayUploadID); e != nil {
		return "", apperror.ErrInvalid
	}
	if _, e := uuid.Parse(in.ThumbnailUploadID); e != nil {
		return "", apperror.ErrInvalid
	}
	if in.TakenAt != nil && in.TakenAt.After(time.Now()) {
		return "", apperror.ErrInvalid
	}
	// Inspect immutable files before opening a transaction.
	var key, mime, provider, class string
	var size int64
	e := s.Pool.QueryRow(ctx, `SELECT storage_key,mime_type,size_bytes,storage_provider,storage_class FROM upload_intents WHERE id=$1 AND owner_id=$2 AND purpose='gallery' AND state='UPLOADED'`, in.DisplayUploadID, user).Scan(&key, &mime, &size, &provider, &class)
	if e != nil {
		return "", db.Error(e)
	}
	meta, e := s.Store.Confirm(ctx, storage.Object{Provider: provider, Class: class, Key: key, MIME: mime, Size: size})
	if e != nil {
		return "", e
	}
	var id string
	e = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.RequirePlatform(ctx, tx, user, "gallery.manage"); e != nil {
			return e
		}
		display, e := upload.Consume(ctx, tx, user, "", in.DisplayUploadID, "gallery")
		if e != nil {
			return e
		}
		thumb, e := upload.Consume(ctx, tx, user, "", in.ThumbnailUploadID, "gallery")
		if e != nil {
			return e
		}
		if thumb.Size > 1<<20 {
			return apperror.ErrInvalid
		}
		if e = tx.QueryRow(ctx, `INSERT INTO gallery_images(uploaded_by,title,caption,alt_text,display_key,thumbnail_key,mime_type,width,height,display_size_bytes,taken_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, user, in.Title, in.Caption, in.AltText, display.Key, thumb.Key, display.MIME, meta.Width, meta.Height, display.Size, in.TakenAt).Scan(&id); e != nil {
			return e
		}
		return audit.Record(ctx, tx, "", user, "GALLERY_IMAGE_CREATED", "gallery", id, nil)
	})
	return id, e
}
func (s Service) Update(ctx context.Context, user, id string, in Update) error {
	if (in.Title != nil && len(*in.Title) > 300) || (in.Caption != nil && len(*in.Caption) > 5000) || (in.AltText != nil && (strings.TrimSpace(*in.AltText) == "" || len(*in.AltText) > 1000)) {
		return apperror.ErrInvalid
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.RequirePlatform(ctx, tx, user, "gallery.manage"); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE gallery_images SET title=COALESCE($2,title),caption=COALESCE($3,caption),alt_text=COALESCE($4,alt_text),updated_at=now() WHERE id=$1 AND status<>'ARCHIVED'`, id, in.Title, in.Caption, in.AltText)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return apperror.ErrNotFound
		}
		return audit.Record(ctx, tx, "", user, "GALLERY_IMAGE_UPDATED", "gallery", id, nil)
	})
}
func (s Service) Transition(ctx context.Context, user, id string, publish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if e := authorization.RequirePlatform(ctx, tx, user, "gallery.manage"); e != nil {
			return e
		}
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM gallery_images WHERE id=$1 FOR UPDATE`, id).Scan(&status); e != nil {
			return e
		}
		target := "ARCHIVED"
		if publish {
			target = "PUBLISHED"
		}
		if status == target {
			return nil
		}
		if publish && status != "DRAFT" {
			return apperror.ErrConflict
		}
		if _, e := tx.Exec(ctx, `UPDATE gallery_images SET status=$2,published_at=CASE WHEN $2='PUBLISHED' THEN now() ELSE published_at END,updated_at=now() WHERE id=$1`, id, target); e != nil {
			return e
		}
		return audit.Record(ctx, tx, "", user, "GALLERY_IMAGE_"+target, "gallery", id, nil)
	})
}
func (s Service) List(ctx context.Context, month, raw string, limit int) (Page, error) {
	p := Page{Data: []Image{}}
	if limit < 1 || limit > 20 {
		return p, apperror.ErrInvalid
	}
	var start, end *time.Time
	if month != "" {
		v, e := time.Parse("2006-01", month)
		if e != nil {
			return p, apperror.ErrInvalid
		}
		w := v.AddDate(0, 1, 0)
		start, end = &v, &w
	}
	var at *time.Time
	var id *string
	if raw != "" {
		c, e := s.decode(raw)
		if e != nil || c.Month != month {
			return p, apperror.ErrInvalid
		}
		at, id = &c.At, &c.ID
	}
	rows, e := s.Pool.Query(ctx, `SELECT g.id,g.title,g.caption,g.alt_text,g.width,g.height,g.taken_at,g.published_at,COALESCE(g.taken_at,g.published_at),g.display_key,g.thumbnail_key,d.storage_provider,d.storage_class,t.storage_provider,t.storage_class FROM gallery_images g JOIN upload_intents d ON d.storage_key=g.display_key JOIN upload_intents t ON t.storage_key=g.thumbnail_key WHERE g.status='PUBLISHED' AND ($1::timestamptz IS NULL OR COALESCE(g.taken_at,g.published_at)>=$1) AND ($2::timestamptz IS NULL OR COALESCE(g.taken_at,g.published_at)<$2) AND ($3::timestamptz IS NULL OR (COALESCE(g.taken_at,g.published_at),g.id)<($3,$4::uuid)) ORDER BY COALESCE(g.taken_at,g.published_at) DESC,g.id DESC LIMIT $5`, start, end, at, id, limit+1)
	if e != nil {
		return p, e
	}
	defer rows.Close()
	var last cursor
	for rows.Next() {
		var i Image
		var date time.Time
		if e = rows.Scan(&i.ID, &i.Title, &i.Caption, &i.AltText, &i.Width, &i.Height, &i.TakenAt, &i.PublishedAt, &date, &i.displayKey, &i.thumbnailKey, &i.displayProvider, &i.displayClass, &i.thumbnailProvider, &i.thumbnailClass); e != nil {
			return p, e
		}
		if len(p.Data) == limit {
			p.NextCursor = s.encode(last)
			break
		}
		s.urls(&i)
		p.Data = append(p.Data, i)
		last = cursor{At: date, ID: i.ID, Month: month}
	}
	return p, rows.Err()
}
func (s Service) urls(i *Image) {
	if i.displayProvider == storage.ProviderR2 && i.displayClass == storage.ClassPublic && i.thumbnailProvider == storage.ProviderR2 && i.thumbnailClass == storage.ClassPublic {
		i.DisplayURL, _ = s.Store.PublicURL(storage.Object{Provider: i.displayProvider, Class: i.displayClass, Key: i.displayKey})
		i.ThumbnailURL, _ = s.Store.PublicURL(storage.Object{Provider: i.thumbnailProvider, Class: i.thumbnailClass, Key: i.thumbnailKey})
		return
	}
	i.DisplayURL = s.BaseURL + "/v1/public/gallery/" + i.ID + "/files/display"
	i.ThumbnailURL = s.BaseURL + "/v1/public/gallery/" + i.ID + "/files/thumbnail"
}
func (s Service) Get(ctx context.Context, id string) (Image, error) {
	var i Image
	e := s.Pool.QueryRow(ctx, `SELECT g.id,g.title,g.caption,g.alt_text,g.width,g.height,g.taken_at,g.published_at,g.display_key,g.thumbnail_key,d.storage_provider,d.storage_class,t.storage_provider,t.storage_class FROM gallery_images g JOIN upload_intents d ON d.storage_key=g.display_key JOIN upload_intents t ON t.storage_key=g.thumbnail_key WHERE g.id=$1 AND g.status='PUBLISHED'`, id).Scan(&i.ID, &i.Title, &i.Caption, &i.AltText, &i.Width, &i.Height, &i.TakenAt, &i.PublishedAt, &i.displayKey, &i.thumbnailKey, &i.displayProvider, &i.displayClass, &i.thumbnailProvider, &i.thumbnailClass)
	s.urls(&i)
	return i, db.Error(e)
}
func (s Service) File(ctx context.Context, id, variant string) (upload.Object, error) {
	var o upload.Object
	if variant != "display" && variant != "thumbnail" {
		return o, apperror.ErrNotFound
	}
	e := s.Pool.QueryRow(ctx, `SELECT i.storage_key,i.file_name,i.mime_type,i.size_bytes,i.storage_provider,i.storage_class FROM gallery_images g JOIN upload_intents i ON i.storage_key=CASE WHEN $2='display' THEN g.display_key ELSE g.thumbnail_key END WHERE g.id=$1 AND g.status='PUBLISHED'`, id, variant).Scan(&o.Key, &o.Name, &o.MIME, &o.Size, &o.Provider, &o.Class)
	return o, db.Error(e)
}
func (s Service) AdminList(ctx context.Context, user, id string, limit, offset int) (json.RawMessage, error) {
	if e := authorization.RequirePlatform(ctx, s.Pool, user, "gallery.manage"); e != nil {
		return nil, e
	}
	b, e := db.JSON(s.Pool.QueryRow(ctx, `SELECT COALESCE(json_agg(t),'[]') FROM(SELECT id,title,caption,alt_text,width,height,mime_type,display_size_bytes,taken_at,published_at,status,created_at FROM gallery_images WHERE ($1='' OR id=NULLIF($1,'')::uuid) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3)t`, id, limit, offset))
	if id != "" {
		return db.One(b, e)
	}
	return b, e
}
func (s Service) encode(c cursor) string {
	b, _ := json.Marshal(c)
	v := base64.RawURLEncoding.EncodeToString(b)
	h := hmac.New(sha256.New, s.Secret)
	_, _ = h.Write([]byte("gallery:" + v))
	return v + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s Service) decode(raw string) (cursor, error) {
	var c cursor
	if len(raw) > 1000 {
		return c, apperror.ErrInvalid
	}
	p := strings.Split(raw, ".")
	if len(p) != 2 {
		return c, apperror.ErrInvalid
	}
	sig, e := base64.RawURLEncoding.DecodeString(p[1])
	if e != nil {
		return c, apperror.ErrInvalid
	}
	h := hmac.New(sha256.New, s.Secret)
	_, _ = h.Write([]byte("gallery:" + p[0]))
	if !hmac.Equal(sig, h.Sum(nil)) {
		return c, apperror.ErrInvalid
	}
	b, e := base64.RawURLEncoding.DecodeString(p[0])
	if e != nil || json.Unmarshal(b, &c) != nil || c.At.IsZero() {
		return c, apperror.ErrInvalid
	}
	if _, e = uuid.Parse(c.ID); e != nil {
		return c, apperror.ErrInvalid
	}
	return c, nil
}
