package storage

import (
	"go.uber.org/multierr"
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
