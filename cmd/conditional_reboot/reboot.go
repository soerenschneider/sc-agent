package deps

import (
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/app"
	"github.com/soerenschneider/sc-agent/pkg/reboot"
)

func BuildRebootImpl(dryRun bool) (app.Reboot, error) {
	if dryRun {
		return &reboot.NoReboot{}, nil
	}

	return &reboot.DefaultRebootImpl{}, nil
}
