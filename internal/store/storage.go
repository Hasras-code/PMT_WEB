package store

import (
	"context"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/student"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents the minimal persisted identity data used by auth and user flows.
type User struct {
	ID            string
	StudentNumber string
	Combination   *student.Combination
	FirstName     string
	LastName      string
	DisplayName   string
	Email         string
	PhoneNumber   *string
	Status        string
	PasswordHash  string
	EmailVerified *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash []byte
	Transport        string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	UserAgent        string
	LastUsedAt       *time.Time
}

type VerificationToken struct {
	ID        string
	UserID    string
	Purpose   string
	TokenHash []byte
	UsedAt    *time.Time
	ExpiresAt time.Time
}

type UserStore interface {
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetStatusByID(ctx context.Context, id string) (string, error)
	LoginData(ctx context.Context, email string) (id, hash, status string, verified *time.Time, err error)
	Create(ctx context.Context, u User, hash string) (string, error)
	Activate(ctx context.Context, id string) error
	UpdatePassword(ctx context.Context, id, hash string) error
	SetStatus(ctx context.Context, id, status string) error
}

type SessionStore interface {
	GetByRefreshHash(ctx context.Context, tokenHash []byte) (Session, error)
	Create(ctx context.Context, session Session) (string, error)
	RevokeByUser(ctx context.Context, userID, sessionID string) error
	RevokeAllByUser(ctx context.Context, userID string) error
}

type VerificationStore interface {
	Create(ctx context.Context, token VerificationToken) error
	Consume(ctx context.Context, tokenHash, purpose string) error
}

type Storage struct {
	Pool *pgxpool.Pool

	Users    UserStore
	Sessions SessionStore
	Tokens   VerificationStore
}

func NewStorage(pool *pgxpool.Pool) Storage {
	return Storage{
		Pool:     pool,
		Users:    &UserRepository{Pool: pool},
		Sessions: &SessionRepository{Pool: pool},
		Tokens:   &VerificationTokenRepository{Pool: pool},
	}
}
