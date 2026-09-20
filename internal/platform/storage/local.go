// Package storage implements private local object storage. Files are not mounted as a static directory.
package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"
	"time"
)

type Local struct {
	root   *os.Root
	secret []byte
}
type Grant struct {
	Key     string `json:"key"`
	ID      string `json:"id"`
	Purpose string `json:"purpose"`
	Name    string `json:"name,omitempty"`
	MIME    string `json:"mime,omitempty"`
	Expiry  int64  `json:"exp"`
}
type Metadata struct {
	Size          int64
	Width, Height int
	MIME          string
}

func Open(dir, secret string) (*Local, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	r, e := os.OpenRoot(dir)
	if e != nil {
		return nil, e
	}
	h := sha256.Sum256([]byte("local-object-capabilities:" + secret))
	return &Local{root: r, secret: h[:]}, nil
}
func (s *Local) Close() error { return s.root.Close() }
func Key() string             { return uuid.NewString() + ".blob" }
func validKey(key string) bool {
	if !strings.HasSuffix(key, ".blob") {
		return false
	}
	u, e := uuid.Parse(strings.TrimSuffix(key, ".blob"))
	return e == nil && u.String()+".blob" == key
}
func (s *Local) Sign(g Grant) (string, error) {
	b, e := json.Marshal(g)
	if e != nil {
		return "", e
	}
	p := base64.RawURLEncoding.EncodeToString(b)
	h := hmac.New(sha256.New, s.secret)
	_, _ = h.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil)), nil
}
func (s *Local) Verify(token, purpose string) (Grant, error) {
	var g Grant
	if len(token) > 4096 {
		return g, apperror.ErrForbidden
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return g, apperror.ErrForbidden
	}
	sig, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return g, apperror.ErrForbidden
	}
	h := hmac.New(sha256.New, s.secret)
	_, _ = h.Write([]byte(parts[0]))
	if !hmac.Equal(sig, h.Sum(nil)) {
		return g, apperror.ErrForbidden
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil || json.Unmarshal(b, &g) != nil || g.Purpose != purpose || g.Expiry <= time.Now().Unix() || !validKey(g.Key) {
		return g, apperror.ErrForbidden
	}
	return g, nil
}
func (s *Local) Read(key string) (*os.File, error) {
	if !validKey(key) {
		return nil, apperror.ErrInvalid
	}
	f, e := s.root.Open(key)
	if os.IsNotExist(e) {
		return nil, apperror.ErrNotFound
	}
	return f, e
}
func (s *Local) Delete(key string) error {
	if !validKey(key) {
		return apperror.ErrInvalid
	}
	e := s.root.Remove(key)
	if os.IsNotExist(e) {
		return nil
	}
	return e
}
func (s *Local) Put(ctx context.Context, key, mime string, size int64, r io.Reader) (Metadata, error) {
	var m Metadata
	if !validKey(key) || size <= 0 || size > 50<<20 {
		return m, apperror.ErrInvalid
	}
	temp := Key() + ".part"
	f, e := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return m, e
	}
	defer f.Close()
	defer func() { _ = s.root.Remove(temp) }()
	n, e := io.Copy(f, io.LimitReader(&contextReader{ctx: ctx, r: r}, size+1))
	if e != nil {
		return m, e
	}
	if n != size {
		return m, apperror.ErrInvalid
	}
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return m, e
	}
	m, e = inspect(f, mime, n)
	if e != nil {
		return m, e
	}
	if e = f.Sync(); e != nil {
		return m, e
	}
	if e = s.root.Link(temp, key); os.IsExist(e) {
		return m, apperror.ErrConflict
	}
	return m, e
}
func (s *Local) Inspect(key, mime string) (Metadata, error) {
	f, e := s.Read(key)
	if e != nil {
		return Metadata{}, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return Metadata{}, e
	}
	return inspect(f, mime, st.Size())
}
func inspect(r io.Reader, mime string, size int64) (Metadata, error) {
	m := Metadata{Size: size, MIME: mime}
	if mime == "application/pdf" {
		b := make([]byte, 5)
		if _, e := io.ReadFull(r, b); e != nil || string(b) != "%PDF-" {
			return m, apperror.ErrInvalid
		}
		return m, nil
	}
	cfg, format, e := image.DecodeConfig(io.LimitReader(r, 1<<20))
	if e != nil {
		return m, apperror.ErrInvalid
	}
	expected := map[string]string{"jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp"}[format]
	if expected != mime || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 16000 || cfg.Height > 16000 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		return m, apperror.ErrInvalid
	}
	m.Width = cfg.Width
	m.Height = cfg.Height
	return m, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(b []byte) (int, error) {
	if e := c.ctx.Err(); e != nil {
		return 0, e
	}
	return c.r.Read(b)
}
func (s *Local) DownloadURL(base, key, name, mime string) (string, error) {
	t, e := s.Sign(Grant{Key: key, Name: name, MIME: mime, Purpose: "download", Expiry: time.Now().Add(5 * time.Minute).Unix()})
	if e != nil {
		return "", e
	}
	return fmt.Sprintf("%s/v1/files/downloads/%s", base, t), nil
}

// CleanupTemporary removes incomplete uploads left by an interrupted process.
func (s *Local) CleanupTemporary(ctx context.Context, age time.Duration) (int, error) {
	dir, e := s.root.Open(".")
	if e != nil {
		return 0, e
	}
	defer dir.Close()
	count := 0
	for {
		if e = ctx.Err(); e != nil {
			return count, e
		}
		entries, err := dir.ReadDir(100)
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".part") || !validKey(strings.TrimSuffix(entry.Name(), ".part")) {
				continue
			}
			info, e := entry.Info()
			if e != nil {
				return count, e
			}
			if time.Since(info.ModTime()) < age {
				continue
			}
			if e = s.root.Remove(entry.Name()); e != nil && !os.IsNotExist(e) {
				return count, e
			}
			count++
			if count >= 1000 {
				return count, nil
			}
		}
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return count, err
		}
	}
}
