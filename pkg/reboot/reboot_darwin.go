package reboot

import (
	"errors"
	"github.com/rs/zerolog/log"
)

type DefaultRebootImpl struct {
}

func (l *DefaultRebootImpl) Reboot() error {
	log.Info().Msgf("Reboot request picked up")
	return errors.New("bla")
}
