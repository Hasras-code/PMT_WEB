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
// domain-specific, stable error code and optional message in the response body.
type Coded struct {
	Base    error
	Code    string
	Message string
}

func (e Coded) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func (e Coded) Unwrap() error { return e.Base }

func WithCode(base error, code string) error { return Coded{Base: base, Code: code} }

func WithCodeAndMessage(base error, code, message string) error {
	return Coded{Base: base, Code: code, Message: message}
}

func Code(err error) string {
	var coded Coded
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}

func Message(err error) string {
	var coded Coded
	if errors.As(err, &coded) && coded.Message != "" {
		return coded.Message
	}
	return ""
}
