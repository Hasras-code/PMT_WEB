package apperror

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("invalid credentials")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict or invalid state")
	ErrInvalid      = errors.New("invalid input")
)

// Coded keeps the API's existing status classification while allowing a
// domain-specific, stable error code in the response body.
type Coded struct {
	Base error
	Code string
}

func (e Coded) Error() string { return e.Code }
func (e Coded) Unwrap() error { return e.Base }

func WithCode(base error, code string) error { return Coded{Base: base, Code: code} }

func Code(err error) string {
	var coded Coded
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}
