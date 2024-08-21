package ports

type SystemPowerStatus interface {
	Shutdown() error
	Reboot() error
}

type MachineShutdownRequest struct {
}
