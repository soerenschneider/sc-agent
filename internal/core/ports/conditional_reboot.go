package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/app"
)

type RebootManager interface {
	Start(ctx context.Context) error
	Pause()
	Status() app.RebootManagerStatus
	Unpause()
	IsPaused() bool
}
