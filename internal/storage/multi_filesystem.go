package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

var (
	ErrFilesNotIdentical = errors.New("files not identical")
	ErrFilesMissing      = errors.New("some files are missing")
)

type MultiFilesystem struct {
	storage []*FilesystemStorage
}

func NewMultiFilesystemStorage(storage ...string) (*MultiFilesystem, error) {
	var storageImpls []*FilesystemStorage
	var errs error
	for _, storageConf := range storage {
		storageImpl, err := NewFilesystemStorageFromUri(storageConf)
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			storageImpls = append(storageImpls, storageImpl)
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &MultiFilesystem{storage: storageImpls}, nil
}

func (fs *MultiFilesystem) Read() ([]byte, error) {
	if err := fs.sanitize(); err != nil {
		if errors.Is(err, ErrFilesMissing) {
			// All defined files have the same checksum but some files are missing. Try to write data to all backends.
			read, err := fs.read()
			if err != nil {
				return nil, err
			}

			// try to write on best-effort
			log.Debug().Str("component", "storage").Msg("trying to write files to all backends")
			if err := fs.Write(read); err != nil {
				log.Error().Err(err).Msg("failed to write data to all configured files")
			}
			return read, nil
		}

		// We found multiple files with different content, no source of truth available
		return nil, err
	}

	return fs.read()
}

// sanitize checks whether all configured storage backends contain data and whether that data is identical, which
// it should be.
// This is needed in cases where a new storage backend is added to the configuration after data was already
// written to existing backends. The new backend would not receive existing data due to the way the Read() method
// is implemented.
func (fs *MultiFilesystem) sanitize() error {
	hashes := map[int]string{}
	for idx := range fs.storage {
		data, err := fs.storage[idx].Read()
		if err == nil {
			hashedBytes := sha256.Sum256(data)
			hashes[idx] = hex.EncodeToString(hashedBytes[:])
		}
	}

	hashesIdentical := true
	for idx := 1; idx < len(hashes); idx++ {
		if hashes[idx] != hashes[0] {
			hashesIdentical = false
		}
	}

	if !hashesIdentical {
		log.Warn().Str("component", "storage").Msg("detected files which differ")
		return ErrFilesNotIdentical
	}

	if len(hashes) > 0 && len(hashes) != len(fs.storage) {
		log.Warn().Str("component", "storage").Msg("not all configured files are present")
		return ErrFilesMissing
	}

	return nil
}

func (fs *MultiFilesystem) read() ([]byte, error) {
	var errs error
	for idx := range fs.storage {
		read, err := fs.storage[idx].Read()
		if err == nil {
			return read, nil
		}
		errs = multierr.Append(errs, err)
	}

	return nil, errs
}

func (fs *MultiFilesystem) CanRead() error {
	var errs error
	for idx := range fs.storage {
		if err := fs.storage[idx].CanRead(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return nil
}

func (fs *MultiFilesystem) Write(data []byte) error {
	var errs error
	for idx := range fs.storage {
		if err := fs.storage[idx].Write(data); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

func (fs *MultiFilesystem) CanWrite() error {
	var errs error
	for idx := range fs.storage {
		if err := fs.storage[idx].CanWrite(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}
