package domain

import "errors"

var ErrSecretReplicationItemNotFound = errors.New("could not find item")

type Formatter interface {
	Format(data map[string]any) ([]byte, error)
}

type SecretReplicationItem struct {
	SecretPath string
	Formatter  Formatter
	DestUri    string
}
