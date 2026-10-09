package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func Open(ctx context.Context, dsn string, max int32) (*pgxpool.Pool, error) {
	c, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		return nil, fmt.Errorf("invalid database configuration")
	}
	c.MaxConns = max
	c.MinConns = 0
	c.MaxConnLifetime = 30 * time.Minute
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	// Supabase transaction pooling can assign a different server connection to
	// each transaction. Use the extended protocol without named prepared
	// statements so cached statement names never depend on a server session.
	c.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	c.ConnConfig.RuntimeParams["statement_timeout"] = "10000"
	c.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "15000"
	p, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return p, nil
}
func Tx(ctx context.Context, p *pgxpool.Pool, f func(pgx.Tx) error) error {
	tx, e := p.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if e = f(tx); e != nil {
		return Error(e)
	}
	return Error(tx.Commit(ctx))
}

// DiagnosticError preserves safe PostgreSQL diagnostics after a database error
// has been normalized for the API. Its Error string stays generic; SQLSTATE and
// operation are intended only for server-side logs.
type DiagnosticError struct {
	Base      error
	SQLState  string
	Operation string
}

func (e *DiagnosticError) Error() string { return e.Base.Error() }
func (e *DiagnosticError) Unwrap() error { return e.Base }

// WithOperation labels a failed database operation without changing its public
// error classification or message.
func WithOperation(err error, operation string) error {
	if err == nil {
		return nil
	}
	var diagnostic *DiagnosticError
	if errors.As(err, &diagnostic) {
		return &DiagnosticError{Base: diagnostic.Base, SQLState: diagnostic.SQLState, Operation: operation}
	}
	return &DiagnosticError{Base: err, Operation: operation}
}

// Diagnostics returns only non-sensitive diagnostic fields for structured logs.
func Diagnostics(err error) (operation, sqlState string) {
	var diagnostic *DiagnosticError
	if errors.As(err, &diagnostic) {
		return diagnostic.Operation, diagnostic.SQLState
	}
	return "", ""
}

func Error(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return apperror.ErrNotFound
	}
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "23505", "23503", "23514", "23P01":
			return apperror.ErrConflict
		case "22P02", "22007", "22008":
			return &DiagnosticError{Base: apperror.ErrInvalid, SQLState: p.Code}
		}
	}
	return e
}

// JSON scans an explicitly selected, safe JSON projection, never a database model.
func JSON(row pgx.Row) ([]byte, error) { var b []byte; e := row.Scan(&b); return b, Error(e) }

// One converts a bounded list projection to a single resource representation.
func One(b []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	var rows []json.RawMessage
	if e := json.Unmarshal(b, &rows); e != nil {
		return nil, e
	}
	if len(rows) == 0 {
		return nil, apperror.ErrNotFound
	}
	return rows[0], nil
}
