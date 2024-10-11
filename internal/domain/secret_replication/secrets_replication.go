package secret_replication

import "errors"

var ErrSecretsReplicationItemNotFound = errors.New("could not find item")

type Formatter interface {
	Format(data map[string]any) ([]byte, error)
}

type StorageImplementation interface {
	Read() ([]byte, error)
	CanRead() error
	Write([]byte) error
}

type SecretReplicationStatus int

const (
	UnknownStatus      SecretReplicationStatus = iota
	FailedStatus       SecretReplicationStatus = iota
	SynchronizedStatus SecretReplicationStatus = iota
)

type ReplicationConf struct {
	Id         string
	SecretPath string
	DestUri    string
}

type ReplicationItem struct {
	ReplicationConf ReplicationConf
	Formatter       Formatter
	Destination     StorageImplementation
	Status          SecretReplicationStatus
}
