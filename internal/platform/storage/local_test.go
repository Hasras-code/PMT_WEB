package storage

import (
	"bytes"
	"context"
	"errors"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"image"
	"image/png"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLocalSecurity(t *testing.T) {
	s, e := Open(t.TempDir(), strings.Repeat("s", 48))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	pdf := []byte("%PDF-1.7\nlocal test")
	key := Key()
	if _, e = s.Put(context.Background(), key, "application/pdf", int64(len(pdf)), bytes.NewReader(pdf)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Put(context.Background(), key, "application/pdf", int64(len(pdf)), bytes.NewReader(pdf)); !errors.Is(e, apperror.ErrConflict) {
		t.Fatalf("overwrite: %v", e)
	}
	f, e := s.Read(key)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := io.ReadAll(f)
	f.Close()
	if !bytes.Equal(b, pdf) {
		t.Fatal("changed bytes")
	}
	for _, key := range []string{"../secret", "/etc/passwd", "a/b.blob", "bad.blob"} {
		if _, e = s.Read(key); e == nil {
			t.Fatal("accepted path", key)
		}
	}
	if _, e = s.Put(context.Background(), Key(), "application/pdf", 3, strings.NewReader("too many")); e == nil {
		t.Fatal("oversize accepted")
	}
	if _, e = s.Put(context.Background(), Key(), "application/pdf", 5, strings.NewReader("<html")); e == nil {
		t.Fatal("spoofed PDF")
	}
	var buf bytes.Buffer
	if e = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 10))); e != nil {
		t.Fatal(e)
	}
	m, e := s.Put(context.Background(), Key(), "image/png", int64(buf.Len()), bytes.NewReader(buf.Bytes()))
	if e != nil || m.Width != 20 || m.Height != 10 {
		t.Fatalf("image: %+v %v", m, e)
	}
	if _, e = s.Put(context.Background(), Key(), "image/jpeg", int64(buf.Len()), bytes.NewReader(buf.Bytes())); e == nil {
		t.Fatal("spoofed image")
	}
	token, e := s.Sign(Grant{Key: key, Purpose: "download", Expiry: time.Now().Add(time.Minute).Unix()})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Verify(token, "download"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Verify(token, "upload"); e == nil {
		t.Fatal("wrong purpose accepted")
	}
	if _, e = s.Verify(token+"x", "download"); e == nil {
		t.Fatal("tamper accepted")
	}
	expired, _ := s.Sign(Grant{Key: key, Purpose: "download", Expiry: time.Now().Add(-time.Minute).Unix()})
	if _, e = s.Verify(expired, "download"); e == nil {
		t.Fatal("expired accepted")
	}
}
func TestConcurrentUploadImmutable(t *testing.T) {
	s, e := Open(t.TempDir(), strings.Repeat("s", 48))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	key := Key()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Put(context.Background(), key, "application/pdf", 8, strings.NewReader("%PDF-1.7"))
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else if !errors.Is(e, apperror.ErrConflict) {
			t.Fatal(e)
		}
	}
	if success != 1 {
		t.Fatalf("successes %d", success)
	}
}
