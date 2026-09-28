package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestOpenUsesTransactionPoolerCompatibleExecMode(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}

	pool, err := Open(context.Background(), dsn, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if got := pool.Config().ConnConfig.DefaultQueryExecMode; got != pgx.QueryExecModeExec {
		t.Fatalf("default query exec mode = %v, want %v", got, pgx.QueryExecModeExec)
	}
}
