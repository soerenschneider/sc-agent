package machine

import (
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
)

const serviceName = "k0s"

type K0sCmd struct {
	useSudo bool
}

func New(conf config.K0s) (*K0sCmd, error) {
	return &K0sCmd{useSudo: conf.UseSudo}, nil
}

func (m *K0sCmd) Stop() error {
	cmd := []string{"k0s", "stop"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Msg("could not stop k0s cluster")
		return err
	}
	return nil
}

func (m *K0sCmd) Start() error {
	cmd := []string{"k0s", "start"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Msg("could not start k0s cluster")
		return err
	}
	return nil
}
