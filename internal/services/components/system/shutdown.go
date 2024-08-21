package system

import (
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
)

const serviceName = "machine"

type MachineCmd struct {
	useSudo bool
}

func New(conf config.PowerStatus) (*MachineCmd, error) {
	return &MachineCmd{useSudo: conf.UseSudo}, nil
}

func (m *MachineCmd) Shutdown() error {
	cmd := []string{"systemctl", "shutdown"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Msg("could not shutdown machine")
		return err
	}
	return nil
}

func (m *MachineCmd) Reboot() error {
	cmd := []string{"systemctl", "reboot"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Msg("could not reboot machine")
		return err
	}
	return nil
}
