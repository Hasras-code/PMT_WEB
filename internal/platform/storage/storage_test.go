package storage

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/aws/smithy-go"
)

func TestR2ObjectKeyAndPublicURL(t *testing.T) {
	r2, err := OpenR2(context.Background(), R2Config{
		Endpoint: "https://account.r2.cloudflarestorage.com", Region: "auto",
		AccessKeyID: "test-access", SecretKey: "test-secret",
		PrivateBucket: "private", PublicBucket: "public", PublicBaseURL: "https://media.example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := &Manager{R2: r2, Provider: ProviderR2, UploadTTL: 10 * time.Minute}
	object, err := manager.NewObject("batches/batch-id/resources", "lecture.pdf", "application/pdf", 1024, ClassPrivate)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(object.Key, "batches/batch-id/resources/") || !strings.HasSuffix(object.Key, ".pdf") || strings.Contains(object.Key, "lecture") {
		t.Fatalf("unsafe or unexpected key %q", object.Key)
	}
	if _, err = manager.NewObject("../escape", "lecture.pdf", "application/pdf", 1, ClassPrivate); err == nil {
		t.Fatal("accepted unsafe prefix")
	}
	if _, err = manager.NewObject("safe", "fake.jpg", "application/pdf", 1, ClassPrivate); err == nil {
		t.Fatal("accepted an extension that does not match the MIME type")
	}

	public := object
	public.Class = ClassPublic
	got, err := r2.PublicURL(public.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://media.example.com/"+public.Key {
		t.Fatalf("public URL = %q", got)
	}
}

func TestR2PresignedUploadUsesConfiguredBucketAndContentType(t *testing.T) {
	r2, err := OpenR2(context.Background(), R2Config{
		Endpoint: "https://account.r2.cloudflarestorage.com", Region: "auto",
		AccessKeyID: "test-access", SecretKey: "test-secret",
		PrivateBucket: "private", PublicBucket: "public", PublicBaseURL: "https://media.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	auth, err := r2.CreateUploadURL(context.Background(), Object{Provider: ProviderR2, Class: ClassPrivate, Key: "batches/a/resources/id.pdf", Name: "notes.pdf", MIME: "application/pdf", Size: 100}, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(auth.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u.Path, "/private/batches/a/resources/id.pdf") {
		t.Fatalf("presigned path = %q", u.Path)
	}
	if auth.Headers["Content-Type"] != "application/pdf" || auth.ExpiresIn != 600 {
		t.Fatalf("upload authorization = %+v", auth)
	}
	public, err := r2.CreateUploadURL(context.Background(), Object{Provider: ProviderR2, Class: ClassPublic, Key: "gallery/id.png", Name: "photo.png", MIME: "image/png", Size: 100}, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	publicURL, err := url.Parse(public.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(publicURL.Path, "/public/gallery/id.png") {
		t.Fatalf("public presigned path = %q", publicURL.Path)
	}
}

func TestPublicURLRejectsUnsafeKey(t *testing.T) {
	if _, err := joinPublicURL("https://media.example.com", "../private/file.pdf"); err == nil {
		t.Fatal("accepted unsafe key")
	}
}

func TestStorageErrorsAreSanitized(t *testing.T) {
	notFound := storageError(&smithy.GenericAPIError{Code: "NoSuchKey", Message: "provider detail"})
	if !errors.Is(notFound, apperror.ErrNotFound) {
		t.Fatalf("not-found mapping = %v", notFound)
	}
	got := storageError(&smithy.GenericAPIError{Code: "InternalError", Message: "sensitive provider detail"})
	if got == nil || strings.Contains(got.Error(), "sensitive") {
		t.Fatalf("storage error leaked provider detail: %v", got)
	}
}
