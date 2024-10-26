package ports

import "github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/app"

type RebootManager interface {
	Start() error
	Pause()
	Status() app.RebootManagerStatus
	Unpause()
	IsPaused() bool
}
