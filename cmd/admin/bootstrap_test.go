package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bootstrapMailbox struct{ token string }

func (m *bootstrapMailbox) Send(_ context.Context, _, _, token string) error {
	m.token = token
	return nil
}

func TestValidateBootstrapConfirmation(t *testing.T) {
	tests := []struct {
		name                string
		database, confirmDB string
		email, confirmEmail string
		wantError           bool
	}{
		{name: "matching confirmations", database: "postgres", confirmDB: "postgres", email: "Admin@example.test", confirmEmail: "admin@example.test"},
		{name: "missing database", confirmDB: "postgres", email: "admin@example.test", confirmEmail: "admin@example.test", wantError: true},
		{name: "database mismatch", database: "postgres", confirmDB: "pmtdb", email: "admin@example.test", confirmEmail: "admin@example.test", wantError: true},
		{name: "email mismatch", database: "postgres", confirmDB: "postgres", email: "admin@example.test", confirmEmail: "other@example.test", wantError: true},
		{name: "missing email confirmation", database: "postgres", confirmDB: "postgres", email: "admin@example.test", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBootstrapConfirmation(tt.database, tt.confirmDB, tt.email, tt.confirmEmail)
			if (err != nil) != tt.wantError {
				t.Fatalf("validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func bootstrapTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "admin_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	migrations, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(migrations)
	for _, migration := range migrations {
		contents, err := os.ReadFile(migration)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("apply %s: %v", migration, err)
		}
	}
	return pool
}

func TestBootstrapPlatformAdminPromotesVerifiedAccountOnce(t *testing.T) {
	pool := bootstrapTestPool(t)
	ctx := context.Background()
	var database string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&database); err != nil {
		t.Fatal(err)
	}
	var batchID string
	if err := pool.QueryRow(ctx, `INSERT INTO batches(name,slug,entry_year) VALUES('Registration cohort','admin-bootstrap',2026) RETURNING id`).Scan(&batchID); err != nil {
		t.Fatal(err)
	}
	mail := &bootstrapMailbox{}
	secret := strings.Repeat("s", 48)
	service := &auth.Service{
		Pool:   pool,
		Signer: auth.Signer{Secret: []byte(secret), Issuer: "test", Audience: "test"},
		Mail:   mail,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Cost:   4,
	}
	input := auth.RegisterInput{
		StudentNumber: "BOOTSTRAP-ADMIN",
		Combination:   "PMT-CS",
		BatchID:       batchID,
		FirstName:     "Initial",
		LastName:      "Administrator",
		DisplayName:   "Initial Administrator",
		Email:         "first-admin@example.test",
		Password:      "correct horse battery",
	}
	if err := service.Register(ctx, input); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM users WHERE lower(email)=lower($1)`, input.Email).Scan(&status); err != nil || status != "PENDING_VERIFICATION" {
		t.Fatalf("new account status = %q, %v", status, err)
	}
	if _, err := bootstrapPlatformAdmin(ctx, pool, database+"-wrong", input.Email); !errors.Is(err, errBootstrapDatabaseMismatch) {
		t.Fatalf("wrong database: got %v", err)
	}
	if _, err := bootstrapPlatformAdmin(ctx, pool, database, input.Email); !errors.Is(err, errBootstrapTargetInvalid) {
		t.Fatalf("unverified account: got %v", err)
	}
	if err := service.ConsumeToken(ctx, mail.token, "EMAIL_VERIFY", ""); err != nil {
		t.Fatal(err)
	}
	var originalHash string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE lower(email)=lower($1)`, input.Email).Scan(&originalHash); err != nil {
		t.Fatal(err)
	}

	userID, err := bootstrapPlatformAdmin(ctx, pool, database, input.Email)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(userID); err != nil {
		t.Fatalf("bootstrap returned invalid user id %q", userID)
	}
	var assigned, audited bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_platform_roles upr
			JOIN roles r ON r.id=upr.role_id AND r.scope='PLATFORM'
			WHERE upr.user_id=$1 AND r.code='PLATFORM_ADMIN'
		), EXISTS (
			SELECT 1 FROM audit_logs
			WHERE action='PLATFORM_ROLE_ASSIGNED' AND entity_type='user'
			AND entity_id=$1 AND actor_user_id IS NULL
			AND new_values @> '{"role":"PLATFORM_ADMIN","source":"cmd/admin bootstrap-platform-admin"}'::jsonb
		)
	`, userID).Scan(&assigned, &audited); err != nil {
		t.Fatal(err)
	}
	if !assigned || !audited {
		t.Fatalf("bootstrap assignment/audit missing: assigned=%v audited=%v", assigned, audited)
	}
	var hashAfter string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, userID).Scan(&hashAfter); err != nil {
		t.Fatal(err)
	}
	if hashAfter != originalHash {
		t.Fatal("bootstrap changed the existing password hash")
	}
	if _, err := service.Login(ctx, input.Email, input.Password, "token", "bootstrap test"); err != nil {
		t.Fatalf("existing credentials could not log in after promotion: %v", err)
	}
	if _, err := bootstrapPlatformAdmin(ctx, pool, database, input.Email); !errors.Is(err, errBootstrapAlreadyUsed) {
		t.Fatalf("repeated bootstrap: got %v", err)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action='PLATFORM_ROLE_ASSIGNED' AND entity_id=$1`, userID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("bootstrap audit count = %d, %v", auditCount, err)
	}
}

func TestBootstrapPlatformAdminRequiresExistingVerifiedUser(t *testing.T) {
	pool := bootstrapTestPool(t)
	ctx := context.Background()
	var database string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&database); err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrapPlatformAdmin(ctx, pool, database, "missing@example.test"); !errors.Is(err, errBootstrapTargetInvalid) {
		t.Fatalf("missing account: got %v", err)
	}
	if _, err := bootstrapPlatformAdmin(ctx, pool, database, "not-an-email"); err == nil {
		t.Fatal("invalid email was accepted")
	}
}

func TestBootstrapPlatformAdminSerializesConcurrentAttempts(t *testing.T) {
	pool := bootstrapTestPool(t)
	ctx := context.Background()
	var database string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&database); err != nil {
		t.Fatal(err)
	}
	var batchID string
	if err := pool.QueryRow(ctx, `INSERT INTO batches(name,slug,entry_year) VALUES('Registration cohort','admin-bootstrap-race',2026) RETURNING id`).Scan(&batchID); err != nil {
		t.Fatal(err)
	}
	mail := &bootstrapMailbox{}
	service := &auth.Service{Pool: pool, Signer: auth.Signer{Secret: []byte(strings.Repeat("s", 48)), Issuer: "test", Audience: "test"}, Mail: mail, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Cost: 4}
	input := auth.RegisterInput{StudentNumber: "BOOTSTRAP-RACE", Combination: "PMT-CS", BatchID: batchID, FirstName: "Race", LastName: "Admin", DisplayName: "Race Admin", Email: "race-admin@example.test", Password: "correct horse battery"}
	if err := service.Register(ctx, input); err != nil {
		t.Fatal(err)
	}
	if err := service.ConsumeToken(ctx, mail.token, "EMAIL_VERIFY", ""); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := bootstrapPlatformAdmin(context.Background(), pool, database, input.Email)
			results <- err
		}()
	}
	close(start)
	var succeeded, rejected int
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, errBootstrapAlreadyUsed):
			rejected++
		default:
			t.Fatalf("unexpected result from concurrent bootstrap: %v", err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("concurrent bootstrap outcomes: succeeded=%d rejected=%d", succeeded, rejected)
	}
}
