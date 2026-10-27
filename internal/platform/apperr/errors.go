package apperr

import "fmt"

var (
	ErrValidation          = fmt.Errorf("validation error")
	ErrUnauthorized        = fmt.Errorf("unauthorized")
	ErrForbidden           = fmt.Errorf("forbidden")
	ErrNotFound            = fmt.Errorf("not found")
	ErrConflict            = fmt.Errorf("conflict")
	ErrIdempotencyConflict = fmt.Errorf("idempotency conflict")
)

type Error struct {
	Kind    error
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func (e Error) Unwrap() error {
	return e.Kind
}

func Validation(msg string) error {
	return Error{Kind: ErrValidation, Message: msg}
}

func Unauthorized(msg string) error {
	return Error{Kind: ErrUnauthorized, Message: msg}
}

func Conflict(msg string) error {
	return Error{Kind: ErrConflict, Message: msg}
}

func NotFound(msg string) error {
	return Error{Kind: ErrNotFound, Message: msg}
}
