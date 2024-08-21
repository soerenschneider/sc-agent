package http_replication

import (
	"errors"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

type Status int

const (
	Unknown         Status = iota
	Synced          Status = iota
	FailedStatus    Status = iota
	InvalidChecksum Status = iota
)

var (
	ErrHttpReplicationItemNotFound = errors.New("could not find item")
	ErrMismatchedHash              = errors.New("mismatched hash")
)

type ReplicationConf struct {
	Id          string
	Source      string
	Destination string
	Sha256Sum   string
}

type ReplicationItem struct {
	ReplicationConf ReplicationConf
	Destination     StorageImplementation
	PostHooks       []domain.PostHook
	Status          Status
}

type StorageImplementation interface {
	Read() ([]byte, error)
	CanRead() error
	Write([]byte) error
}
