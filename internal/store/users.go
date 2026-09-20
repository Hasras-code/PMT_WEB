package store

import (
	"context"
	"errors"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	Pool *pgxpool.Pool
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.Pool.QueryRow(ctx, `SELECT id, student_number, first_name, last_name, display_name, email, phone_number, status, created_at, updated_at FROM users WHERE id=$1`, id).Scan(
		&u.ID,
		&u.StudentNumber,
		&u.FirstName,
		&u.LastName,
		&u.DisplayName,
		&u.Email,
		&u.PhoneNumber,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apperror.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.Pool.QueryRow(ctx, `SELECT id, student_number, first_name, last_name, display_name, email, phone_number, status, password_hash, email_verified_at, created_at, updated_at FROM users WHERE lower(email)=$1`, email).Scan(
		&u.ID,
		&u.StudentNumber,
		&u.FirstName,
		&u.LastName,
		&u.DisplayName,
		&u.Email,
		&u.PhoneNumber,
		&u.Status,
		&u.PasswordHash,
		&u.EmailVerified,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apperror.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) GetStatusByID(ctx context.Context, id string) (string, error) {
	var status string
	err := r.Pool.QueryRow(ctx, `SELECT status FROM users WHERE id=$1`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperror.ErrNotFound
	}
	return status, err
}

func (r *UserRepository) LoginData(ctx context.Context, email string) (id, hash, status string, verified *time.Time, err error) {
	var u User
	err = r.Pool.QueryRow(ctx, `SELECT id, password_hash, status, email_verified_at FROM users WHERE lower(email)=$1`, email).Scan(&u.ID, &u.PasswordHash, &u.Status, &u.EmailVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", nil, apperror.ErrNotFound
	}
	if err != nil {
		return "", "", "", nil, err
	}
	return u.ID, u.PasswordHash, u.Status, u.EmailVerified, nil
}

func (r *UserRepository) Create(ctx context.Context, u User, hash string) (string, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `INSERT INTO users(student_number,first_name,last_name,display_name,email,password_hash) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING RETURNING id`, u.StudentNumber, u.FirstName, u.LastName, u.DisplayName, u.Email, hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperror.ErrConflict
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, hash string) error {
	result, err := r.Pool.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, id, hash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Activate(ctx context.Context, id string) error {
	result, err := r.Pool.Exec(ctx, `UPDATE users SET status='ACTIVE', email_verified_at=now(), updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *UserRepository) SetStatus(ctx context.Context, id, status string) error {
	result, err := r.Pool.Exec(ctx, `UPDATE users SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
