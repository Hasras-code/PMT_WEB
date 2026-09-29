package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("storage migration failed", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.R2Endpoint == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretKey == "" || cfg.R2PrivateBucket == "" || cfg.R2PublicBucket == "" || cfg.R2PublicBaseURL == "" {
		return fmt.Errorf("R2 configuration is required")
	}
	dryRun := len(os.Args) > 1 && os.Args[1] == "--dry-run"
	if len(os.Args) > 1 && !dryRun {
		return fmt.Errorf("usage: migrate-storage [--dry-run]")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	pool, err := db.Open(ctx, cfg.DatabaseURL, 2)
	if err != nil {
		return err
	}
	defer pool.Close()
	local, err := storage.Open(cfg.StorageDir, cfg.Secret)
	if err != nil {
		return err
	}
	defer local.Close()
	r2, err := storage.OpenR2(ctx, storage.R2Config{
		Endpoint: cfg.R2Endpoint, Region: cfg.R2Region,
		AccessKeyID: cfg.R2AccessKeyID, SecretKey: cfg.R2SecretKey,
		PrivateBucket: cfg.R2PrivateBucket, PublicBucket: cfg.R2PublicBucket,
		PublicBaseURL: cfg.R2PublicBaseURL,
	})
	if err != nil {
		return err
	}
	if err = r2.Check(ctx); err != nil {
		return err
	}

	rows, err := pool.Query(ctx, `SELECT id,storage_key,file_name,mime_type,size_bytes,storage_class FROM upload_intents WHERE storage_provider='LOCAL' AND state IN ('UPLOADED','CONSUMED') ORDER BY created_at,id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var migrated, verified, missing int
	referenced := make(map[string]struct{})
	for rows.Next() {
		var id string
		object := storage.Object{Provider: storage.ProviderR2}
		if err = rows.Scan(&id, &object.Key, &object.Name, &object.MIME, &object.Size, &object.Class); err != nil {
			return err
		}
		referenced[object.Key] = struct{}{}
		file, openErr := local.Read(object.Key)
		if openErr != nil {
			missing++
			log.Warn("local object missing", "upload_id", id, "storage_key", object.Key)
			continue
		}
		if dryRun {
			_ = file.Close()
			verified++
			continue
		}

		_, inspectErr := r2.Inspect(ctx, object)
		switch {
		case inspectErr == nil:
			verified++
		case errors.Is(inspectErr, apperror.ErrNotFound):
			if _, err = r2.Put(ctx, object, file); err != nil {
				_ = file.Close()
				return fmt.Errorf("upload %s: %w", id, err)
			}
			migrated++
		default:
			_ = file.Close()
			return fmt.Errorf("verify existing R2 object %s: %w", id, inspectErr)
		}
		if err = file.Close(); err != nil {
			return err
		}
		tag, err := pool.Exec(ctx, `UPDATE upload_intents SET storage_provider='R2' WHERE id=$1 AND storage_provider='LOCAL'`, id)
		if err != nil {
			return fmt.Errorf("mark %s migrated: %w", id, err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("upload %s changed during migration", id)
		}
		log.Info("object migrated", "upload_id", id, "storage_key", object.Key, "storage_class", strings.ToLower(object.Class))
	}
	if err = rows.Err(); err != nil {
		return err
	}
	entries, err := os.ReadDir(cfg.StorageDir)
	if err != nil {
		return err
	}
	orphans := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".part") {
			continue
		}
		if _, ok := referenced[entry.Name()]; !ok {
			orphans++
			log.Warn("unreferenced local object retained", "storage_key", entry.Name())
		}
	}
	log.Info("storage migration completed", "dry_run", dryRun, "uploaded", migrated, "already_verified", verified, "missing_local_files", missing, "unreferenced_local_files", orphans)
	if missing > 0 {
		return fmt.Errorf("%d referenced local objects are missing", missing)
	}
	return nil
}
