package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/google/uuid"
)

const (
	ProviderLocal = "LOCAL"
	ProviderR2    = "R2"
	ClassPrivate  = "PRIVATE"
	ClassPublic   = "PUBLIC"
)

type Object struct {
	Provider string
	Class    string
	Key      string
	Name     string
	MIME     string
	Size     int64
}

type UploadAuthorization struct {
	URL       string
	Headers   map[string]string
	ExpiresIn int64
}

type Manager struct {
	Local       *Local
	R2          *R2
	Provider    string
	BaseURL     string
	UploadTTL   time.Duration
	DownloadTTL time.Duration
}

type Settings struct {
	Provider                  string
	LocalDir, Secret, BaseURL string
	UploadTTL, DownloadTTL    time.Duration
	R2                        R2Config
}

func OpenManager(ctx context.Context, settings Settings) (*Manager, error) {
	local, err := Open(settings.LocalDir, settings.Secret)
	if err != nil {
		return nil, err
	}
	manager := &Manager{Local: local, Provider: strings.ToUpper(settings.Provider), BaseURL: settings.BaseURL, UploadTTL: settings.UploadTTL, DownloadTTL: settings.DownloadTTL}
	if manager.UploadTTL == 0 {
		manager.UploadTTL = 10 * time.Minute
	}
	if manager.DownloadTTL == 0 {
		manager.DownloadTTL = 5 * time.Minute
	}
	r2Configured := settings.R2.Endpoint != "" && settings.R2.AccessKeyID != "" && settings.R2.SecretKey != "" && settings.R2.PrivateBucket != "" && settings.R2.PublicBucket != ""
	if manager.Provider == ProviderR2 || r2Configured {
		manager.R2, err = OpenR2(ctx, settings.R2)
		if err != nil {
			_ = local.Close()
			return nil, err
		}
	}
	return manager, nil
}

func (m *Manager) NewObject(prefix, fileName, mime string, size int64, class string) (Object, error) {
	provider := strings.ToUpper(m.Provider)
	if provider == "" {
		provider = ProviderLocal
	}
	if class != ClassPrivate && class != ClassPublic {
		return Object{}, apperror.ErrInvalid
	}
	key := Key()
	if provider == ProviderR2 {
		ext := strings.ToLower(filepath.Ext(fileName))
		if !safeExtension(ext, mime) {
			return Object{}, apperror.ErrInvalid
		}
		prefix = strings.Trim(prefix, "/")
		if !validPrefix(prefix) {
			return Object{}, apperror.ErrInvalid
		}
		key = path.Join(prefix, uuid.NewString()+ext)
	}
	return Object{Provider: provider, Class: class, Key: key, Name: fileName, MIME: mime, Size: size}, nil
}

func (m *Manager) CreateUpload(ctx context.Context, object Object, uploadID string) (UploadAuthorization, error) {
	switch object.Provider {
	case ProviderLocal:
		token, err := m.Local.Sign(Grant{Key: object.Key, ID: uploadID, Purpose: "upload", Expiry: time.Now().Add(m.UploadTTL).Unix()})
		if err != nil {
			return UploadAuthorization{}, err
		}
		return UploadAuthorization{URL: m.BaseURL + "/v1/files/uploads/" + token, Headers: map[string]string{"Content-Type": object.MIME}, ExpiresIn: int64(m.UploadTTL.Seconds())}, nil
	case ProviderR2:
		if m.R2 == nil {
			return UploadAuthorization{}, fmt.Errorf("R2 storage is not configured")
		}
		return m.R2.CreateUploadURL(ctx, object, m.UploadTTL)
	default:
		return UploadAuthorization{}, fmt.Errorf("unsupported storage provider")
	}
}

func (m *Manager) Confirm(ctx context.Context, object Object) (Metadata, error) {
	switch object.Provider {
	case ProviderLocal:
		return m.Local.Inspect(object.Key, object.MIME)
	case ProviderR2:
		if m.R2 == nil {
			return Metadata{}, fmt.Errorf("R2 storage is not configured")
		}
		return m.R2.Inspect(ctx, object)
	default:
		return Metadata{}, fmt.Errorf("unsupported storage provider")
	}
}

func (m *Manager) DownloadURL(ctx context.Context, object Object) (string, error) {
	switch object.Provider {
	case ProviderLocal:
		return m.Local.DownloadURL(m.BaseURL, object.Key, object.Name, object.MIME)
	case ProviderR2:
		if m.R2 == nil {
			return "", fmt.Errorf("R2 storage is not configured")
		}
		return m.R2.CreateDownloadURL(ctx, object, m.DownloadTTL)
	default:
		return "", fmt.Errorf("unsupported storage provider")
	}
}

func (m *Manager) PublicURL(object Object) (string, error) {
	if object.Provider != ProviderR2 || object.Class != ClassPublic {
		return "", apperror.ErrInvalid
	}
	if m.R2 == nil {
		return "", fmt.Errorf("R2 storage is not configured")
	}
	return m.R2.PublicURL(object.Key)
}

func (m *Manager) Delete(ctx context.Context, object Object) error {
	switch object.Provider {
	case ProviderLocal:
		return m.Local.Delete(object.Key)
	case ProviderR2:
		if m.R2 == nil {
			return fmt.Errorf("R2 storage is not configured")
		}
		return m.R2.Delete(ctx, object)
	default:
		return fmt.Errorf("unsupported storage provider")
	}
}

func (m *Manager) Put(ctx context.Context, object Object, reader io.Reader) (Metadata, error) {
	switch object.Provider {
	case ProviderLocal:
		return m.Local.Put(ctx, object.Key, object.MIME, object.Size, reader)
	case ProviderR2:
		if m.R2 == nil {
			return Metadata{}, fmt.Errorf("R2 storage is not configured")
		}
		return m.R2.Put(ctx, object, reader)
	default:
		return Metadata{}, fmt.Errorf("unsupported storage provider")
	}
}

func (m *Manager) VerifyLocal(token, purpose string) (Grant, error) {
	if m.Local == nil {
		return Grant{}, apperror.ErrNotFound
	}
	return m.Local.Verify(token, purpose)
}

func (m *Manager) Close() error {
	if m.Local != nil {
		return m.Local.Close()
	}
	return nil
}

func validPrefix(prefix string) bool {
	return prefix != "" && !strings.HasPrefix(prefix, "/") && !strings.Contains(prefix, "\\") && !strings.Contains(prefix, "..")
}

func validObjectKey(key string) bool {
	return key != "" && len(key) <= 1024 && !strings.HasPrefix(key, "/") && !strings.Contains(key, "\\") && !strings.Contains(key, "..")
}

func safeExtension(ext, mime string) bool {
	for _, allowed := range map[string][]string{
		"application/pdf": {".pdf"},
		"image/jpeg":      {".jpg", ".jpeg"},
		"image/png":       {".png"},
		"image/webp":      {".webp"},
	}[mime] {
		if ext == allowed {
			return true
		}
	}
	return false
}

func joinPublicURL(base, key string) (string, error) {
	if !validObjectKey(key) {
		return "", apperror.ErrInvalid
	}
	u, err := url.Parse(strings.TrimRight(base, "/") + "/" + strings.TrimLeft(key, "/"))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("invalid public storage URL")
	}
	return u.String(), nil
}
