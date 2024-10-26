package app

import (
	"errors"
	"time"
)

func SafeMinSystemUptime(duration time.Duration) ConditionalRebootOpts {
	return func(c *ConditionalReboot) error {
		if duration.Hours() <= 1 {
			return errors.New("duration should not be less than 1 hour")
		}

		c.safeMinSystemUptime = duration
		return nil
	}
}

func DryRun() ConditionalRebootOpts {
	return func(c *ConditionalReboot) error {
		c.ignoreRebootRequests.Store(true)
		return nil
	}
}
