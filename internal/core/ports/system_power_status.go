package ports

import "github.com/soerenschneider/sc-agent/internal/domain/system"

type SystemPowerStatus interface {
	SetCpuGovernor(governor system.Governor) error
	GetCurrentCpuGovernor() (string, error)
	Shutdown() error
	Reboot() error
}

type MachineShutdownRequest struct {
}
