package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}
type Signer struct {
	Secret           []byte
	Issuer, Audience string
	Now              func() time.Time
}

func (s Signer) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func (s Signer) Sign(user, session string) (string, error) {
	now := s.now()
	c := Claims{SessionID: session, RegisteredClaims: jwt.RegisteredClaims{Subject: user, Issuer: s.Issuer, Audience: jwt.ClaimStrings{s.Audience}, IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.Secret)
}
func (s Signer) Parse(raw string) (Claims, error) {
	var c Claims
	_, e := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) { return s.Secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(s.Issuer), jwt.WithAudience(s.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(s.now))
	if e != nil {
		return c, apperror.ErrUnauthorized
	}
	if _, e = uuid.Parse(c.Subject); e != nil {
		return c, apperror.ErrUnauthorized
	}
	if _, e = uuid.Parse(c.SessionID); e != nil || c.IssuedAt == nil {
		return c, apperror.ErrUnauthorized
	}
	return c, nil
}
func RandomToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", nil, e
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	return s, Hash(s), nil
}
func Hash(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }
