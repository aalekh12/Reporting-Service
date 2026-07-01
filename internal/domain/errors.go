package domain

import "fmt"

// ErrorKind classifies a domain error so handlers can map it to an HTTP
// status without knowing anything about the underlying cause.
type ErrorKind string

const (
	KindValidation ErrorKind = "validation"
	KindNotFound   ErrorKind = "not_found"
	KindInternal   ErrorKind = "internal"
)

// Error is the single error type returned by usecase/querybuilder/repository
// code. Wrap lower-level errors with Wrap so the kind survives up to the
// handler layer.
type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func NewValidationError(msg string, args ...any) *Error {
	return &Error{Kind: KindValidation, Message: fmt.Sprintf(msg, args...)}
}

func NewNotFoundError(msg string, args ...any) *Error {
	return &Error{Kind: KindNotFound, Message: fmt.Sprintf(msg, args...)}
}

func WrapInternal(err error, msg string, args ...any) *Error {
	return &Error{Kind: KindInternal, Message: fmt.Sprintf(msg, args...), Err: err}
}
