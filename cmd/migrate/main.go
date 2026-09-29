package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	rawDSN := os.Getenv("DATABASE_URL")
	if rawDSN == "" {
		return fmt.Errorf("DATABASE_URL required")
	}
	dsn, err := migrationURL(rawDSN)
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	m, e := migrate.New("file://migrations", dsn)
	if e != nil {
		return e
	}
	defer m.Close()
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	switch direction {
	case "up":
		e = m.Up()
	case "down":
		e = m.Steps(-1)
	default:
		return fmt.Errorf("usage: migrate [up|down]")
	}
	if errors.Is(e, migrate.ErrNoChange) {
		return nil
	}
	return e
}

func migrationURL(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, ".pooler.supabase.com") && u.Port() == "6543" {
		return "", fmt.Errorf("supabase transaction pooler port 6543 is unsafe for migration advisory locks; use the session pooler on port 5432 or a direct connection")
	}
	if strings.HasSuffix(host, ".supabase.co") || strings.HasSuffix(host, ".pooler.supabase.com") {
		switch strings.ToLower(u.Query().Get("sslmode")) {
		case "require", "verify-ca", "verify-full":
		default:
			return "", fmt.Errorf("supabase migrations require sslmode=require or stronger")
		}
	}

	u.Scheme = "pgx5"
	query := u.Query()
	query.Set("default_query_exec_mode", "exec")
	u.RawQuery = query.Encode()
	return u.String(), nil
}
