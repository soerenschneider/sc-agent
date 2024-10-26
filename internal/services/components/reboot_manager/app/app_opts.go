package app

import (
	"errors"
	"time"
)

func SafeMinSystemUptime(duration time.Duration) RebootManagerOpts {
	return func(c *RebootManager) error {
		if duration.Hours() <= 1 {
			return errors.New("duration should not be less than 1 hour")
		}

		c.safeMinSystemUptime = duration
		return nil
	}
}

func DryRun() RebootManagerOpts {
	return func(c *RebootManager) error {
		c.ignoreRebootRequests.Store(true)
		return nil
	}
}
