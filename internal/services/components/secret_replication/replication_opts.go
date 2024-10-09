package secret_replication

import (
	"errors"
	"time"
)

func WithSyncInterval(interval time.Duration) func(syncer *Service) error {
	return func(s *Service) error {
		if interval.Seconds() < 60 || interval.Hours() > 12 {
			return errors.New("sync interval should be [60s, 12h]")
		}

		s.replicationInterval = interval
		return nil
	}
}
