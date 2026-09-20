package main

import (
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"os"
	"strings"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL required")
	}
	dsn = strings.Replace(dsn, "postgres://", "pgx5://", 1)
	dsn = strings.Replace(dsn, "postgresql://", "pgx5://", 1)
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
