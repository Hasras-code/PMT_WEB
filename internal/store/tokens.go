package store

import (
	"context"
	"errors"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VerificationTokenRepository struct {
	Pool *pgxpool.Pool
}

func (r *VerificationTokenRepository) Create(ctx context.Context, token VerificationToken) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO verification_tokens(user_id, purpose, token_hash, expires_at) VALUES($1,$2,$3,$4)`, token.UserID, token.Purpose, token.TokenHash, token.ExpiresAt)
	return err
}

func (r *VerificationTokenRepository) Consume(ctx context.Context, tokenHash, purpose string) error {
	var uid string
	err := r.Pool.QueryRow(ctx, `SELECT user_id FROM verification_tokens WHERE token_hash=$1 AND purpose=$2 AND used_at IS NULL AND expires_at>now() FOR UPDATE`, tokenHash, purpose).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.ErrInvalid
	}
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `UPDATE verification_tokens SET used_at=now() WHERE user_id=$1 AND purpose=$2 AND used_at IS NULL`, uid, purpose)
	return err
}

var _ time.Time
