package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWTValidation(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	s := Signer{Secret: []byte("01234567890123456789012345678901234567890123456789"), Issuer: "issuer", Audience: "audience", Now: func() time.Time { return now }}
	u, sid := uuid.NewString(), uuid.NewString()
	raw, e := s.Sign(u, sid)
	if e != nil {
		t.Fatal(e)
	}
	if c, e := s.Parse(raw); e != nil || c.Subject != u || c.SessionID != sid {
		t.Fatalf("valid token: %v", e)
	}
	for _, name := range []string{"expired", "issuer", "audience", "algorithm", "signature", "missing_exp", "bad_subject", "bad_session", "future_iat"} {
		t.Run(name, func(t *testing.T) {
			c := Claims{SessionID: sid, RegisteredClaims: jwt.RegisteredClaims{Subject: u, Issuer: s.Issuer, Audience: jwt.ClaimStrings{s.Audience}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}}
			alg := jwt.SigningMethodHS256
			secret := s.Secret
			switch name {
			case "expired":
				c.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Minute))
			case "issuer":
				c.Issuer = "other"
			case "audience":
				c.Audience = jwt.ClaimStrings{"other"}
			case "algorithm":
				alg = jwt.SigningMethodHS384
			case "signature":
				secret = []byte("other-secret")
			case "missing_exp":
				c.ExpiresAt = nil
			case "bad_subject":
				c.Subject = "abc"
			case "bad_session":
				c.SessionID = "abc"
			case "future_iat":
				c.IssuedAt = jwt.NewNumericDate(now.Add(time.Hour))
			}
			raw, e := jwt.NewWithClaims(alg, c).SignedString(secret)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.Parse(raw); e == nil {
				t.Fatal("accepted invalid token")
			}
		})
	}
}
func TestPasswordLimits(t *testing.T) {
	for _, x := range []struct {
		p     string
		valid bool
	}{{"12345678901", false}, {"123456789012", true}, {string(make([]byte, 73)), false}, {"界界界界界界界界界界界界", true}} {
		if PasswordValid(x.p) != x.valid {
			t.Errorf("password length %d", len(x.p))
		}
	}
}
