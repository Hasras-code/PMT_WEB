package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Mailer interface {
	Send(context.Context, string, string, string) error
}
type Service struct {
	Pool   *pgxpool.Pool
	Store  store.Storage
	Signer Signer
	Mail   Mailer
	Log    *slog.Logger
	Cost   int
}
type RegisterInput struct {
	StudentNumber string `json:"student_number"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	Password      string `json:"password"`
}
type User struct {
	ID            string  `json:"id"`
	StudentNumber string  `json:"student_number"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	DisplayName   string  `json:"display_name"`
	Email         string  `json:"email"`
	PhoneNumber   *string `json:"phone_number"`
	Status        string  `json:"status"`
}
type Tokens struct {
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token,omitempty"`
	ExpiresIn      int       `json:"expires_in"`
	SessionID      string    `json:"session_id"`
	Transport      string    `json:"-"`
	RefreshExpires time.Time `json:"-"`
}

func PasswordValid(p string) bool { return utf8.RuneCountInString(p) >= 8 && len(p) <= 72 }
func EmailValid(s string) bool {
	a, e := mail.ParseAddress(s)
	return e == nil && a.Address == s && len(s) <= 254
}
func (s *Service) password(p string) (string, error) {
	if !PasswordValid(p) {
		return "", apperror.ErrInvalid
	}
	cost := s.Cost
	if cost == 0 {
		cost = 12
	}
	b, e := bcrypt.GenerateFromPassword([]byte(p), cost)
	return string(b), e
}
func (s *Service) Register(ctx context.Context, in RegisterInput) error {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !EmailValid(in.Email) || len(in.StudentNumber) < 2 || len(in.StudentNumber) > 64 || strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" || strings.TrimSpace(in.DisplayName) == "" || len(in.FirstName) > 100 || len(in.LastName) > 100 || len(in.DisplayName) > 100 {
		return apperror.ErrInvalid
	}
	hash, e := s.password(in.Password)
	if e != nil {
		return e
	}
	raw, digest, e := RandomToken()
	if e != nil {
		return e
	}
	e = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var id string
		e := tx.QueryRow(ctx, `INSERT INTO users(student_number,first_name,last_name,display_name,email,password_hash) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING RETURNING id`, in.StudentNumber, in.FirstName, in.LastName, in.DisplayName, in.Email, hash).Scan(&id)
		if errors.Is(e, pgx.ErrNoRows) {
			return apperror.ErrConflict
		}
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO verification_tokens(user_id,purpose,token_hash,expires_at) VALUES($1,'EMAIL_VERIFY',$2,now()+interval '24 hours')`, id, digest)
		return e
	})
	if errors.Is(e, apperror.ErrConflict) {
		return nil
	}
	if e != nil {
		return e
	}
	s.send(ctx, in.Email, "EMAIL_VERIFY", raw)
	return nil
}
func (s *Service) send(ctx context.Context, email, purpose, token string) {
	if e := s.Mail.Send(ctx, email, purpose, token); e != nil {
		s.Log.Warn("mail delivery failed; resend is available", "purpose", purpose)
	}
}
func (s *Service) RequestToken(ctx context.Context, email, purpose string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !EmailValid(email) {
		return apperror.ErrInvalid
	}
	raw, hash, e := RandomToken()
	if e != nil {
		return e
	}
	sent := false
	e = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var id, status string
		e := tx.QueryRow(ctx, `SELECT id,status FROM users WHERE lower(email)=$1 FOR UPDATE`, email).Scan(&id, &status)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		if (purpose == "EMAIL_VERIFY" && status != "PENDING_VERIFICATION") || (purpose == "PASSWORD_RESET" && status != "ACTIVE") {
			return nil
		}
		_, e = tx.Exec(ctx, `UPDATE verification_tokens SET used_at=now() WHERE user_id=$1 AND purpose=$2 AND used_at IS NULL`, id, purpose)
		if e != nil {
			return e
		}
		ttl := time.Hour * 24
		if purpose == "PASSWORD_RESET" {
			ttl = 30 * time.Minute
		}
		_, e = tx.Exec(ctx, `INSERT INTO verification_tokens(user_id,purpose,token_hash,expires_at) VALUES($1,$2,$3,$4)`, id, purpose, hash, time.Now().Add(ttl))
		sent = e == nil
		return e
	})
	if e == nil && sent {
		s.send(ctx, email, purpose, raw)
	}
	return e
}
func (s *Service) ConsumeToken(ctx context.Context, raw, purpose, password string) error {
	if len(raw) != 43 {
		return apperror.ErrInvalid
	}
	var passwordHash string
	var e error
	if purpose == "PASSWORD_RESET" {
		passwordHash, e = s.password(password)
		if e != nil {
			return e
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var uid string
		if e := tx.QueryRow(ctx, `SELECT user_id FROM verification_tokens WHERE token_hash=$1 AND purpose=$2`, Hash(raw), purpose).Scan(&uid); e != nil {
			return apperror.ErrInvalid
		}
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM users WHERE id=$1 FOR UPDATE`, uid).Scan(&status); e != nil {
			return e
		}
		var id string
		if e := tx.QueryRow(ctx, `SELECT id FROM verification_tokens WHERE token_hash=$1 AND purpose=$2 AND used_at IS NULL AND expires_at>now() FOR UPDATE`, Hash(raw), purpose).Scan(&id); e != nil {
			return apperror.ErrInvalid
		}
		if purpose == "EMAIL_VERIFY" {
			if status != "PENDING_VERIFICATION" {
				return apperror.ErrInvalid
			}
			_, e = tx.Exec(ctx, `UPDATE users SET status='ACTIVE',email_verified_at=now(),updated_at=now() WHERE id=$1`, uid)
		} else {
			if status != "ACTIVE" {
				return apperror.ErrInvalid
			}
			_, e = tx.Exec(ctx, `UPDATE users SET password_hash=$2,updated_at=now() WHERE id=$1`, uid, passwordHash)
			if e != nil {
				return e
			}
			_, e = tx.Exec(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`, uid)
		}
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE verification_tokens SET used_at=now() WHERE user_id=$1 AND purpose=$2 AND used_at IS NULL`, uid, purpose)
		return e
	})
}

// This valid hash makes unknown-user login perform a bcrypt comparison too.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), 12)

func (s *Service) Login(ctx context.Context, email, password, transport, agent string) (Tokens, error) {
	var out Tokens
	if transport == "" {
		transport = "token"
	}
	if transport != "token" && transport != "cookie" {
		return out, apperror.ErrInvalid
	}
	if len(password) > 72 {
		return out, apperror.ErrUnauthorized
	}
	var id, hash, status string
	var verified *time.Time
	e := s.Pool.QueryRow(ctx, `SELECT id,password_hash,status,email_verified_at FROM users WHERE lower(email)=$1`, strings.ToLower(strings.TrimSpace(email))).Scan(&id, &hash, &status, &verified)
	if errors.Is(e, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return out, apperror.ErrUnauthorized
	}
	if e != nil {
		return out, e
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || status != "ACTIVE" || verified == nil {
		return out, apperror.ErrUnauthorized
	}
	raw, digest, e := RandomToken()
	if e != nil {
		return out, e
	}
	expires := time.Now().Add(30 * 24 * time.Hour)
	e = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var currentHash, currentStatus string
		if e := tx.QueryRow(ctx, `SELECT password_hash,status FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&currentHash, &currentStatus); e != nil {
			return e
		}
		if currentHash != hash || currentStatus != "ACTIVE" {
			return apperror.ErrUnauthorized
		}
		return tx.QueryRow(ctx, `INSERT INTO auth_sessions(user_id,refresh_token_hash,transport,expires_at,user_agent) VALUES($1,$2,$3,$4,$5) RETURNING id`, id, digest, transport, expires, truncate(agent, 512)).Scan(&out.SessionID)
	})
	if e != nil {
		return out, e
	}
	out.AccessToken, e = s.Signer.Sign(id, out.SessionID)
	out.RefreshToken = raw
	out.ExpiresIn = 900
	out.Transport = transport
	out.RefreshExpires = expires
	return out, e
}
func (s *Service) Refresh(ctx context.Context, raw, transport string) (Tokens, error) {
	var out Tokens
	if len(raw) != 43 {
		return out, apperror.ErrUnauthorized
	}
	hash := Hash(raw)
	next, digest, e := RandomToken()
	if e != nil {
		return out, e
	}
	replay := false
	e = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var sid, uid string
		e := tx.QueryRow(ctx, `SELECT id,user_id FROM auth_sessions WHERE refresh_token_hash=$1 UNION ALL SELECT s.id,s.user_id FROM spent_refresh_tokens h JOIN auth_sessions s ON s.id=h.session_id WHERE h.token_hash=$1 LIMIT 1`, hash).Scan(&sid, &uid)
		if errors.Is(e, pgx.ErrNoRows) {
			return apperror.ErrUnauthorized
		}
		if e != nil {
			return e
		}
		var status string
		if e = tx.QueryRow(ctx, `SELECT status FROM users WHERE id=$1 FOR UPDATE`, uid).Scan(&status); e != nil {
			return e
		}
		var current []byte
		var revoked *time.Time
		var mode string
		var expiry time.Time
		if e = tx.QueryRow(ctx, `SELECT refresh_token_hash,revoked_at,expires_at,transport FROM auth_sessions WHERE id=$1 FOR UPDATE`, sid).Scan(&current, &revoked, &expiry, &mode); e != nil {
			return e
		}
		if mode != transport || status != "ACTIVE" || revoked != nil || !expiry.After(time.Now()) {
			return apperror.ErrUnauthorized
		}
		if !equalHash(current, hash) {
			_, e = tx.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE id=$1`, sid)
			replay = true
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO spent_refresh_tokens(token_hash,session_id) VALUES($1,$2)`, hash, sid); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE auth_sessions SET refresh_token_hash=$2,last_used_at=now() WHERE id=$1`, sid, digest); e != nil {
			return e
		}
		out.AccessToken, e = s.Signer.Sign(uid, sid)
		out.SessionID = sid
		out.RefreshToken = next
		out.ExpiresIn = 900
		out.Transport = mode
		out.RefreshExpires = expiry
		return e
	})
	if replay {
		return Tokens{}, apperror.ErrUnauthorized
	}
	return out, e
}
func (s *Service) Authenticate(ctx context.Context, raw string) (Claims, error) {
	c, e := s.Signer.Parse(raw)
	if e != nil {
		return c, e
	}
	var ok bool
	e = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_sessions s JOIN users u ON u.id=s.user_id WHERE s.id=$1 AND s.user_id=$2 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.status='ACTIVE')`, c.SessionID, c.Subject).Scan(&ok)
	if e != nil {
		return c, e
	}
	if !ok {
		return c, apperror.ErrUnauthorized
	}
	return c, nil
}
func (s *Service) Logout(ctx context.Context, raw, transport string) error {
	_, e := s.Pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE transport=$2 AND (refresh_token_hash=$1 OR id IN (SELECT session_id FROM spent_refresh_tokens WHERE token_hash=$1))`, Hash(raw), transport)
	return e
}
func (s *Service) Revoke(ctx context.Context, user, session string) error {
	q := `UPDATE auth_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`
	args := []any{user}
	if session != "" {
		q += ` AND id=$2`
		args = append(args, session)
	}
	tag, e := s.Pool.Exec(ctx, q, args...)
	if e != nil {
		return e
	}
	if session != "" && tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
func (s *Service) Me(ctx context.Context, id string) (User, error) {
	if s.Store.Users != nil {
		u, e := s.Store.Users.GetByID(ctx, id)
		if e == nil {
			return User{ID: u.ID, StudentNumber: u.StudentNumber, FirstName: u.FirstName, LastName: u.LastName, DisplayName: u.DisplayName, Email: u.Email, PhoneNumber: u.PhoneNumber, Status: u.Status}, nil
		}
		if !errors.Is(e, apperror.ErrNotFound) {
			return User{}, e
		}
	}
	var u User
	e := s.Pool.QueryRow(ctx, `SELECT id,student_number,first_name,last_name,display_name,email,phone_number,status FROM users WHERE id=$1`, id).Scan(&u.ID, &u.StudentNumber, &u.FirstName, &u.LastName, &u.DisplayName, &u.Email, &u.PhoneNumber, &u.Status)
	return u, db.Error(e)
}
func equalHash(a, b []byte) bool { return subtle.ConstantTimeCompare(a, b) == 1 }
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
