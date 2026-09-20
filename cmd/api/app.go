package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/httpapi"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/mail"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
	"github.com/Hasras-code/PMT_WEB.git/internal/store"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	cfg     config.Config
	pool    *pgxpool.Pool
	store   store.Storage
	auth    *auth.Service
	files   *storage.Local
	uploads upload.Service
	logger  *slog.Logger
}

func newApp(ctx context.Context, logger *slog.Logger) (*app, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL, cfg.MaxConns)
	if err != nil {
		return nil, err
	}

	files, err := storage.Open(cfg.StorageDir, cfg.Secret)
	if err != nil {
		pool.Close()
		return nil, err
	}

	store := store.NewStorage(pool)
	as := &auth.Service{
		Pool:  pool,
		Store: store,
		Signer: auth.Signer{
			Secret:   []byte(cfg.Secret),
			Issuer:   cfg.Issuer,
			Audience: cfg.Audience,
		},
		Mail: mail.SMTP{Addr: cfg.SMTP, From: cfg.MailFrom},
		Log:  logger,
	}

	return &app{
		cfg:     cfg,
		pool:    pool,
		store:   store,
		auth:    as,
		files:   files,
		uploads: upload.Service{Pool: pool, Store: files, BaseURL: cfg.BaseURL},
		logger:  logger,
	}, nil
}

func (a *app) close() {
	if a != nil {
		if a.pool != nil {
			a.pool.Close()
		}
		if a.files != nil {
			a.files.Close()
		}
	}
}

func (a *app) mount() http.Handler {
	api := &httpapi.API{
		Pool:    a.pool,
		Auth:    a.auth,
		Files:   a.files,
		Uploads: a.uploads,
		Config:  a.cfg,
		Log:     a.logger,
	}
	return api.Router()
}

func (a *app) run() error {
	server := &http.Server{
		Addr:              a.cfg.Addr,
		Handler:           a.mount(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       90 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan error, 1)
	go func() {
		a.logger.Info("API listening", "address", a.cfg.Addr)
		done <- server.ListenAndServe()
	}()

	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}
