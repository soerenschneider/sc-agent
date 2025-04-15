package http_replication

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type Status int

const (
	Unknown          Status = iota
	Synced           Status = iota
	FailedStatus     Status = iota
	ValidationFailed Status = iota
)

var (
	ErrHttpReplicationItemNotFound = errors.New("could not find item")
	ErrFileValidationFailed        = errors.New("file validation failed")
)

type ReplicationConf struct {
	Id              string
	Source          string
	Destinations    []string
	TrimWhitespaces bool
	FileValidation  *FileValidation
}

type FileValidation struct {
	InvertResult bool
	Test         string
	Arg          string
}

func (f *FileValidation) Accept(value []byte) (bool, error) {
	var result bool
	var err error

	switch f.Test {
	case config.FileValidationTestSha256:
		hash := sha256.Sum256(value)
		hashStr := hex.EncodeToString(hash[:])
		result = hashStr == f.Arg

	case config.FileValidationTestRegex:
		var re *regexp.Regexp
		re, err = regexp.Compile(f.Arg)
		if err != nil {
			return false, fmt.Errorf("invalid regex: %w", err)
		}
		result = re.Match(value)

	case config.FileValidationTestStartsWith:
		result = bytes.HasPrefix(value, []byte(f.Arg))

	case config.FileValidationTestEndsWith:
		result = bytes.HasSuffix(value, []byte(f.Arg))

	default:
		return false, fmt.Errorf("unknown validation type %s", f.Test)
	}

	// Apply inversion if needed
	if f.InvertResult {
		result = !result
	}

	return result, nil
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
