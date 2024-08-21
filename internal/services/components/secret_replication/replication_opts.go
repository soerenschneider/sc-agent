package secret_replication

import (
	"errors"
	"time"

	"github.com/spf13/afero"
)

func WithAferoFs(fs afero.Fs) func(syncer *Service) error {
	return func(s *Service) error {
		if fs == nil {
			return errors.New("empty fs provided")
		}

		s.fsImpl = fs
		return nil
	}
}

func WithSyncInterval(interval time.Duration) func(syncer *Service) error {
	return func(s *Service) error {
		if interval.Seconds() < 60 || interval.Hours() > 12 {
			return errors.New("sync interval should be [60s, 12h]")
		}

		s.replicationInterval = interval
		return nil
	}
}
