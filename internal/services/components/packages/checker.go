package packages

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type PackageManager interface {
	CheckUpdate() (domain.CheckUpdateResult, error)
}

type UpdatesAvailableChecker struct {
	packageImpl PackageManager
}

func NewUpdatesAvailableChecker(packageManagerImpl PackageManager) (*UpdatesAvailableChecker, error) {
	return &UpdatesAvailableChecker{
		packageImpl: packageManagerImpl,
	}, nil
}

func (s *UpdatesAvailableChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)

	_, err := s.packageImpl.CheckUpdate()
	if err != nil {
		log.Error().Str("component", "package-updatechecker").Err(err).Msg("error while checking for updates")
	}

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			_, err := s.packageImpl.CheckUpdate()
			if err != nil {
				log.Error().Str("component", "package-updatechecker").Err(err).Msg("error while checking for updates")
			}
		}
	}
}
