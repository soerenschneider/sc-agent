package domain

import "errors"

var (
	ErrPermissionDenied     = errors.New("action not allowed")
	ErrVaultInvalidResponse = errors.New("invalid vault response")
	ErrNotImplemented       = errors.New("not implemented")
	ErrComponentDisabled    = errors.New("component disabled")
)
