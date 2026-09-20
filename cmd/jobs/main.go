package main

import (
	"context"
	"github.com/Hasras-code/PMT_WEB.git/internal/notification"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"log/slog"
	"os"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if e := run(log); e != nil {
		log.Error("jobs failed", "error", e)
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	c, e := config.Load()
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	p, e := db.Open(ctx, c.DatabaseURL, 2)
	if e != nil {
		return e
	}
	defer p.Close()
	s, e := storage.Open(c.StorageDir, c.Secret)
	if e != nil {
		return e
	}
	defer s.Close()
	n, e := (notification.Service{Pool: p}).Dispatch(ctx)
	if e != nil {
		return e
	}
	m, e := upload.Cleanup(ctx, p, s)
	if e != nil {
		return e
	}
	if _, e = p.Exec(ctx, `DELETE FROM auth_sessions WHERE expires_at<now()-interval '30 days'`); e != nil {
		return e
	}
	if _, e = p.Exec(ctx, `DELETE FROM verification_tokens WHERE expires_at<now()-interval '30 days'`); e != nil {
		return e
	}
	if _, e = s.CleanupTemporary(ctx, 24*time.Hour); e != nil {
		return e
	}
	log.Info("jobs completed", "notification_events", n, "expired_uploads", m)
	return nil
}
