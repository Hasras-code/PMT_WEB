package main

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/Hasras-code/PMT_WEB.git/internal/audit"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errBootstrapDatabaseMismatch = errors.New("connected database does not match the explicitly confirmed database")
	errBootstrapAlreadyUsed      = errors.New("platform administrator bootstrap is only available before any platform administrator is assigned")
	errBootstrapTargetInvalid    = errors.New("target must be an existing active, email-verified user")
)

func validateBootstrapConfirmation(database, confirmDatabase, email, confirmEmail string) error {
	if strings.TrimSpace(database) == "" || confirmDatabase != database || strings.TrimSpace(email) == "" || confirmEmail == "" || !strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(confirmEmail)) {
		return errors.New("database and target email confirmations must match")
	}
	return nil
}

// bootstrapPlatformAdmin promotes an existing, verified account exactly once.
// It intentionally never creates an identity or changes a password.
func bootstrapPlatformAdmin(ctx context.Context, pool *pgxpool.Pool, expectedDatabase, email string) (string, error) {
	expectedDatabase = strings.TrimSpace(expectedDatabase)
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if expectedDatabase == "" || err != nil || address.Address != email {
		return "", errors.New("a database name and valid target email are required")
	}

	var userID string
	err = db.Tx(ctx, pool, func(tx pgx.Tx) error {
		var actualDatabase string
		if err := tx.QueryRow(ctx, `SELECT current_database()`).Scan(&actualDatabase); err != nil {
			return err
		}
		if actualDatabase != expectedDatabase {
			return errBootstrapDatabaseMismatch
		}

		// Serialize first-admin attempts, including attempts made by separate
		// CLI processes, before checking the global assignment state.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('pmt.bootstrap-platform-admin:' || current_database(), 0))`); err != nil {
			return err
		}

		var assigned bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM user_platform_roles upr
				JOIN roles r ON r.id=upr.role_id AND r.scope='PLATFORM'
				WHERE r.code='PLATFORM_ADMIN'
			)
		`).Scan(&assigned); err != nil {
			return err
		}
		if assigned {
			return errBootstrapAlreadyUsed
		}

		if err := tx.QueryRow(ctx, `
			SELECT id
			FROM users
			WHERE lower(email)=$1 AND status='ACTIVE' AND email_verified_at IS NOT NULL
			FOR UPDATE
		`, email).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errBootstrapTargetInvalid
			}
			return err
		}

		var roleID string
		if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE code='PLATFORM_ADMIN' AND scope='PLATFORM'`).Scan(&roleID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("platform administrator role is missing; apply the authorization migrations first")
			}
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_platform_roles(user_id,role_id,scope,assigned_by) VALUES($1,$2,'PLATFORM',NULL)`, userID, roleID); err != nil {
			return err
		}
		return audit.Record(ctx, tx, "", "", "PLATFORM_ROLE_ASSIGNED", "user", userID, map[string]string{
			"role":     "PLATFORM_ADMIN",
			"source":   "cmd/admin bootstrap-platform-admin",
			"database": actualDatabase,
		})
	})
	if err != nil {
		return "", fmt.Errorf("bootstrap platform administrator: %w", err)
	}
	return userID, nil
}
