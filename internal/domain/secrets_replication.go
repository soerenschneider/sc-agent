package domain

import "errors"

var ErrSecretsReplicationItemNotFound = errors.New("could not find item")

type Formatter interface {
	Format(data map[string]any) ([]byte, error)
}

type SecretReplicationStatus int

const (
	UnknownStatus      SecretReplicationStatus = iota
	FailedStatus       SecretReplicationStatus = iota
	SynchronizedStatus SecretReplicationStatus = iota
)

type SecretReplicationItem struct {
	Id         string
	SecretPath string
	Formatter  Formatter
	DestUri    string
	Status     SecretReplicationStatus
}
