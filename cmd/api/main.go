package main

import (
	"context"
	"errors"
	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/httpapi"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/mail"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if e := run(log); e != nil {
		log.Error("server stopped", "error", e)
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	c, e := config.Load()
	if e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, e := db.Open(ctx, c.DatabaseURL, c.MaxConns)
	if e != nil {
		return e
	}
	defer pool.Close()
	files, e := storage.Open(c.StorageDir, c.Secret)
	if e != nil {
		return e
	}
	defer files.Close()
	as := &auth.Service{Pool: pool, Signer: auth.Signer{Secret: []byte(c.Secret), Issuer: c.Issuer, Audience: c.Audience}, Mail: mail.SMTP{Addr: c.SMTP, From: c.MailFrom}, Log: log}
	api := &httpapi.API{Pool: pool, Auth: as, Files: files, Uploads: upload.Service{Pool: pool, Store: files, BaseURL: c.BaseURL}, Config: c, Log: log}
	server := &http.Server{Addr: c.Addr, Handler: api.Router(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 90 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	done := make(chan error, 1)
	go func() { log.Info("API listening", "address", c.Addr); done <- server.ListenAndServe() }()
	select {
	case e = <-done:
		if errors.Is(e, http.ErrServerClosed) {
			return nil
		}
		return e
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if e = server.Shutdown(shutdown); e != nil {
			_ = server.Close()
			return e
		}
		return nil
	}
}
