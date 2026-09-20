package httpapi

import (
	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

func (a *API) authRoutes(r chi.Router) {
	r.Post("/auth/register", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[auth.RegisterInput](w, r)
		if e != nil {
			return e
		}
		if e = a.Auth.Register(r.Context(), in); e != nil {
			return e
		}
		return accepted(w)
	}))
	for _, x := range []struct{ path, purpose string }{{"verify-email", "EMAIL_VERIFY"}, {"reset-password", "PASSWORD_RESET"}} {
		r.Post("/auth/"+x.path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			in, e := decode[struct {
				Token    string `json:"token"`
				Password string `json:"password"`
			}](w, r)
			if e != nil {
				return e
			}
			if e = a.Auth.ConsumeToken(r.Context(), in.Token, x.purpose, in.Password); e != nil {
				return e
			}
			return send(w, 204, nil)
		}))
	}
	for _, x := range []struct{ path, purpose string }{{"resend-verification", "EMAIL_VERIFY"}, {"forgot-password", "PASSWORD_RESET"}} {
		r.Post("/auth/"+x.path, a.wrap(func(w http.ResponseWriter, r *http.Request) error {
			in, e := decode[struct {
				Email string `json:"email"`
			}](w, r)
			if e != nil {
				return e
			}
			if e = a.Auth.RequestToken(r.Context(), in.Email, x.purpose); e != nil {
				return e
			}
			return accepted(w)
		}))
	}
	r.Post("/auth/login", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[struct {
			Email     string `json:"email"`
			Password  string `json:"password"`
			Transport string `json:"transport"`
		}](w, r)
		if e != nil {
			return e
		}
		if in.Transport == "cookie" && !a.origin(r.Header.Get("Origin")) {
			return apperror.ErrForbidden
		}
		t, e := a.Auth.Login(r.Context(), in.Email, in.Password, in.Transport, r.UserAgent())
		if e != nil {
			return e
		}
		return a.tokens(w, t)
	}))
	r.Post("/auth/refresh", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		raw, mode, e := a.credential(w, r)
		if e != nil {
			return e
		}
		t, e := a.Auth.Refresh(r.Context(), raw, mode)
		if e != nil {
			return e
		}
		return a.tokens(w, t)
	}))
	r.Post("/auth/logout", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		raw, mode, e := a.credential(w, r)
		if e != nil {
			return e
		}
		if e = a.Auth.Logout(r.Context(), raw, mode); e != nil {
			return e
		}
		if mode == "cookie" {
			a.cookie(w, "", time.Unix(1, 0), -1)
		}
		return send(w, 204, nil)
	}))
}
func accepted(w http.ResponseWriter) error {
	return send(w, 202, map[string]string{"message": "If eligible, an email will be sent with the next step"})
}
func (a *API) cookie(w http.ResponseWriter, value string, expires time.Time, max int) {
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: value, Path: "/v1/auth", HttpOnly: true, Secure: a.Config.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: max})
}
func (a *API) tokens(w http.ResponseWriter, t auth.Tokens) error {
	if t.Transport == "cookie" {
		a.cookie(w, t.RefreshToken, t.RefreshExpires, int(time.Until(t.RefreshExpires).Seconds()))
		t.RefreshToken = ""
	}
	return send(w, 200, t)
}
func (a *API) credential(w http.ResponseWriter, r *http.Request) (string, string, error) {
	var raw string
	if r.ContentLength != 0 {
		in, e := decode[struct {
			RefreshToken string `json:"refresh_token"`
		}](w, r)
		if e != nil {
			return "", "", e
		}
		raw = in.RefreshToken
	}
	c, e := r.Cookie("refresh_token")
	if e == nil {
		if raw != "" || !a.origin(r.Header.Get("Origin")) {
			return "", "", apperror.ErrForbidden
		}
		return c.Value, "cookie", nil
	}
	if raw == "" {
		return "", "", apperror.ErrUnauthorized
	}
	return raw, "token", nil
}
func (a *API) meRoutes(r chi.Router) {
	s := user.Service{Pool: a.Pool}
	r.Post("/auth/logout-all", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := a.Auth.Revoke(r.Context(), userID(r), ""); e != nil {
			return e
		}
		a.cookie(w, "", time.Unix(1, 0), -1)
		return send(w, 204, nil)
	}))
	r.Get("/me", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		v, e := a.Auth.Me(r.Context(), userID(r))
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Patch("/me/profile", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		in, e := decode[user.Profile](w, r)
		if e != nil {
			return e
		}
		if e = s.Update(r.Context(), userID(r), in); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/me/sessions", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.Sessions(r.Context(), userID(r), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Delete("/me/sessions/{sessionID}", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := a.Auth.Revoke(r.Context(), userID(r), param(r, "sessionID")); e != nil {
			return e
		}
		return send(w, 204, nil)
	}))
	r.Get("/me/bookmarks", a.wrap(func(w http.ResponseWriter, r *http.Request) error {
		l, o, e := page(r)
		if e != nil {
			return e
		}
		v, e := s.Bookmarks(r.Context(), userID(r), l, o)
		if e != nil {
			return e
		}
		return send(w, 200, v)
	}))
	r.Post("/me/profile/image/uploads", a.uploadHandler("profile", ""))
}
