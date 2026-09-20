package store

import (
	"context"
	"errors"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	Pool *pgxpool.Pool
}

func (r *SessionRepository) GetByRefreshHash(ctx context.Context, tokenHash []byte) (Session, error) {
	var s Session
	err := r.Pool.QueryRow(ctx, `SELECT id, user_id, refresh_token_hash, transport, expires_at, revoked_at, user_agent, last_used_at FROM auth_sessions WHERE refresh_token_hash=$1 LIMIT 1`, tokenHash).Scan(
		&s.ID,
		&s.UserID,
		&s.RefreshTokenHash,
		&s.Transport,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.UserAgent,
		&s.LastUsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, apperror.ErrNotFound
	}
	return s, err
}

func (r *SessionRepository) Create(ctx context.Context, session Session) (string, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `INSERT INTO auth_sessions(user_id, refresh_token_hash, transport, expires_at, user_agent) VALUES($1,$2,$3,$4,$5) RETURNING id`, session.UserID, session.RefreshTokenHash, session.Transport, session.ExpiresAt, truncate(session.UserAgent, 512)).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *SessionRepository) RevokeByUser(ctx context.Context, userID, sessionID string) error {
	result, err := r.Pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at, now()) WHERE user_id=$1 AND id=$2`, userID, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) RevokeAllByUser(ctx context.Context, userID string) error {
	_, err := r.Pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at, now()) WHERE user_id=$1`, userID)
	return err
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

var _ time.Time
