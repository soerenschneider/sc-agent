package reboot

type NoReboot struct {
}

func (d *NoReboot) Reboot() error {
	return nil
}
