package apperror

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("invalid credentials")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict or invalid state")
	ErrInvalid      = errors.New("invalid input")
)
