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
			return apperror.ErrInvalid
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
