package ports

import "github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/app"

type ConditionalReboot interface {
	Start() error
	Pause()
	Status() app.ConditionalRebootStatus
	Unpause()
	IsPaused() bool
}
