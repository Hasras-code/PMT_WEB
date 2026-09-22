package httpapi

import (
	"net/http"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/Hasras-code/PMT_WEB.git/internal/auth"
	"github.com/Hasras-code/PMT_WEB.git/internal/user"
	"github.com/go-chi/chi/v5"
)

func (a *API) authRoutes(r chi.Router) {
	r.Post("/auth/register", a.wrap(a.registerHandler))
	r.Post("/auth/verify-email", a.wrap(a.verifyEmailHandler))
	r.Post("/auth/reset-password", a.wrap(a.resetPasswordHandler))
	r.Post("/auth/resend-verification", a.wrap(a.resendVerificationHandler))
	r.Post("/auth/forgot-password", a.wrap(a.forgotPasswordHandler))
	r.Post("/auth/login", a.wrap(a.loginHandler))
	r.Post("/auth/refresh", a.wrap(a.refreshHandler))
	r.Post("/auth/logout", a.wrap(a.logoutHandler))
}

// registerHandler godoc
//
//	@Summary		Register a new user
//	@Description	Creates a pending account for the selected cohort and sends an email verification message. Verification activates the membership with the STUDENT role.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	auth.RegisterInput	true	"Registration details"
//	@Success		202		{object}	map[string]string
//	@Failure		400,422	{object}	map[string]any
//	@Router			/v1/auth/register [post]
func (a *API) registerHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[auth.RegisterInput](w, r)
	if e != nil {
		return e
	}
	if e = a.Auth.Register(r.Context(), in); e != nil {
		return e
	}
	return accepted(w)
}

// verifyEmailHandler godoc
//
//	@Summary		Verify an email address
//	@Description	Activates a pending account and adds it to its selected cohort with the STUDENT role.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	tokenInput	true	"Verification token"
//	@Success		204
//	@Failure		422	{object}	map[string]any
//	@Router			/v1/auth/verify-email [post]
func (a *API) verifyEmailHandler(w http.ResponseWriter, r *http.Request) error {
	return a.consumeTokenHandler(w, r, "EMAIL_VERIFY")
}

// resetPasswordHandler godoc
//
//	@Summary		Reset a password
//	@Description	Sets a new password using a password reset token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	resetPasswordInput	true	"Reset details"
//	@Success		204
//	@Failure		422	{object}	map[string]any
//	@Router			/v1/auth/reset-password [post]
func (a *API) resetPasswordHandler(w http.ResponseWriter, r *http.Request) error {
	return a.consumeTokenHandler(w, r, "PASSWORD_RESET")
}

func (a *API) consumeTokenHandler(w http.ResponseWriter, r *http.Request, purpose string) error {
	in, e := decode[resetPasswordInput](w, r)
	if e != nil {
		return e
	}
	if e = a.Auth.ConsumeToken(r.Context(), in.Token, purpose, in.Password); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// resendVerificationHandler godoc
//
//	@Summary		Resend email verification
//	@Description	Requests another verification email without revealing account state.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	emailInput	true	"Email address"
//	@Success		202		{object}	map[string]string
//	@Router			/v1/auth/resend-verification [post]
func (a *API) resendVerificationHandler(w http.ResponseWriter, r *http.Request) error {
	return a.requestTokenHandler(w, r, "EMAIL_VERIFY")
}

// forgotPasswordHandler godoc
//
//	@Summary		Request a password reset
//	@Description	Requests a password reset email without revealing account state.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	emailInput	true	"Email address"
//	@Success		202		{object}	map[string]string
//	@Router			/v1/auth/forgot-password [post]
func (a *API) forgotPasswordHandler(w http.ResponseWriter, r *http.Request) error {
	return a.requestTokenHandler(w, r, "PASSWORD_RESET")
}

func (a *API) requestTokenHandler(w http.ResponseWriter, r *http.Request, purpose string) error {
	in, e := decode[emailInput](w, r)
	if e != nil {
		return e
	}
	if e = a.Auth.RequestToken(r.Context(), in.Email, purpose); e != nil {
		return e
	}
	return accepted(w)
}

// loginHandler godoc
//
//	@Summary		Log in
//	@Description	Authenticates with email and password using token or browser-cookie transport.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	loginInput	true	"Login credentials"
//	@Success		200		{object}	auth.Tokens
//	@Failure		401	{object}	map[string]any
//	@Failure		403	{object}	map[string]any
//	@Router			/v1/auth/login [post]
func (a *API) loginHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[loginInput](w, r)
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
}

// refreshHandler godoc
//
//	@Summary		Refresh an access token
//	@Description	Rotates the refresh credential and issues a new access token.
//	@Tags			auth
//	@Produce		json
//	@Success		200		{object}	auth.Tokens
//	@Failure		401	{object}	map[string]any
//	@Router			/v1/auth/refresh [post]
func (a *API) refreshHandler(w http.ResponseWriter, r *http.Request) error {
	raw, mode, e := a.credential(w, r)
	if e != nil {
		return e
	}
	t, e := a.Auth.Refresh(r.Context(), raw, mode)
	if e != nil {
		return e
	}
	return a.tokens(w, t)
}

// logoutHandler godoc
//
//	@Summary		Log out
//	@Description	Revokes the current refresh session.
//	@Tags			auth
//	@Produce		json
//	@Success		204
//	@Failure		401	{object}	map[string]any
//	@Router			/v1/auth/logout [post]
func (a *API) logoutHandler(w http.ResponseWriter, r *http.Request) error {
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
}

type emailInput struct {
	Email string `json:"email"`
}

type tokenInput struct {
	Token string `json:"token"`
}

type resetPasswordInput struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type loginInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Transport string `json:"transport"`
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
	r.Post("/auth/logout-all", a.wrap(a.logoutAllHandler))
	r.Get("/me", a.wrap(a.meHandler))
	r.Patch("/me/profile", a.wrap(a.updateProfileHandler))
	r.Get("/me/sessions", a.wrap(a.sessionsHandler))
	r.Delete("/me/sessions/{sessionID}", a.wrap(a.revokeSessionHandler))
	r.Get("/me/bookmarks", a.wrap(a.bookmarksHandler))
	r.Get("/me/access", a.wrap(a.accessHandler))
	r.Post("/me/profile/image/uploads", a.wrap(a.authorizeProfileUploadHandler))
}

// accessHandler godoc
//
//	@Summary Get current authorization context
//	@Description Returns platform permissions and active batch roles and permissions for the authenticated user.
//	@Tags users
//	@Produce json
//	@Success 200 {object} map[string]any
//	@Security bearerAuth
//	@Router /v1/me/access [get]
func (a *API) accessHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := (user.Service{Pool: a.Pool}).Access(r.Context(), userID(r))
	if e != nil {
		return e
	}
	return send(w, http.StatusOK, v)
}

// logoutAllHandler godoc
//
//	@Summary		Log out all sessions
//	@Description	Revokes every active session for the current user.
//	@Tags			auth
//	@Success		204
//	@Security		bearerAuth
//	@Router			/v1/auth/logout-all [post]
func (a *API) logoutAllHandler(w http.ResponseWriter, r *http.Request) error {
	if e := a.Auth.Revoke(r.Context(), userID(r), ""); e != nil {
		return e
	}
	a.cookie(w, "", time.Unix(1, 0), -1)
	return send(w, 204, nil)
}

// meHandler godoc
//
//	@Summary		Get the current user
//	@Description	Returns the authenticated user's account.
//	@Tags			users
//	@Produce		json
//	@Success		200		{object}	auth.User
//	@Security		bearerAuth
//	@Router			/v1/me [get]
func (a *API) meHandler(w http.ResponseWriter, r *http.Request) error {
	v, e := a.Auth.Me(r.Context(), userID(r))
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// updateProfileHandler godoc
//
//	@Summary		Update the current profile
//	@Description	Updates profile fields for the authenticated user.
//	@Tags			users
//	@Accept			json
//	@Param			payload	body	user.Profile	true	"Profile details"
//	@Success		204
//	@Security		bearerAuth
//	@Router			/v1/me/profile [patch]
func (a *API) updateProfileHandler(w http.ResponseWriter, r *http.Request) error {
	in, e := decode[user.Profile](w, r)
	if e != nil {
		return e
	}
	if e = (user.Service{Pool: a.Pool}).Update(r.Context(), userID(r), in); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// sessionsHandler godoc
//
//	@Summary		List sessions
//	@Description	Returns the authenticated user's active sessions.
//	@Tags			users
//	@Produce		json
//	@Param			limit	query	int	false	"Limit"
//	@Param			offset	query	int	false	"Offset"
//	@Success		200		{array}	store.Session
//	@Security		bearerAuth
//	@Router			/v1/me/sessions [get]
func (a *API) sessionsHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (user.Service{Pool: a.Pool}).Sessions(r.Context(), userID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}

// revokeSessionHandler godoc
//
//	@Summary		Revoke a session
//	@Description	Revokes one session belonging to the current user.
//	@Tags			users
//	@Param			sessionID	path	string	true	"Session ID"
//	@Success		204
//	@Security		bearerAuth
//	@Router			/v1/me/sessions/{sessionID} [delete]
func (a *API) revokeSessionHandler(w http.ResponseWriter, r *http.Request) error {
	if e := a.Auth.Revoke(r.Context(), userID(r), param(r, "sessionID")); e != nil {
		return e
	}
	return send(w, 204, nil)
}

// bookmarksHandler godoc
//
//	@Summary		List bookmarks
//	@Description	Returns the authenticated user's bookmarks.
//	@Tags			users
//	@Produce		json
//	@Param			limit	query	int	false	"Limit"
//	@Param			offset	query	int	false	"Offset"
//	@Success		200		{array}	map[string]any
//	@Security		bearerAuth
//	@Router			/v1/me/bookmarks [get]
func (a *API) bookmarksHandler(w http.ResponseWriter, r *http.Request) error {
	l, o, e := page(r)
	if e != nil {
		return e
	}
	v, e := (user.Service{Pool: a.Pool}).Bookmarks(r.Context(), userID(r), l, o)
	if e != nil {
		return e
	}
	return send(w, 200, v)
}
