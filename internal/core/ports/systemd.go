package ports

type Systemd interface {
	Restart(unit string) error
	Logs(req SystemdLogsRequest) ([]string, error)
}

type SystemdRestartUnitRequest struct {
	Unit string `json:"unit" validate:"required"`
}

type SystemdLogsRequest struct {
	Unit  string `json:"unit" validate:"required"`
	Lines int    `json:"lines" validate:"gt=0,lt=1000"`
}
