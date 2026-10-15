package domain

import "errors"

var (
	ErrPermissionDenied     = errors.New("action not allowed")
	ErrVaultInvalidResponse = errors.New("invalid vault response")
	ErrNotImplemented       = errors.New("not implemented")
	ErrComponentDisabled    = errors.New("component disabled")

	// ErrNotFound and ErrInvalidInput categorize errors caused by the caller, independent of the transport. Their
	// messages are safe to be returned to the caller.
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

// categorizedError is an error with its own message that also matches the given category via errors.Is.
type categorizedError struct {
	msg      string
	category error
}

func (e *categorizedError) Error() string {
	return e.msg
}

func (e *categorizedError) Unwrap() error {
	return e.category
}

func newCategorizedError(category error, msg string) error {
	return &categorizedError{msg: msg, category: category}
}
