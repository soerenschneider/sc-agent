package libvirt

import (
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
)

const serviceName = "libvirt"

type LibvirtCmd struct {
	useSudo bool
}

func New(conf config.Libvirt) (*LibvirtCmd, error) {
	ret := &LibvirtCmd{}

	return ret, nil
}

func (l *LibvirtCmd) RebootDomain(domain string) error {
	cmd := []string{
		"virsh", "reboot", domain,
	}

	var c *exec.Cmd
	if l.useSudo {
		c = exec.Command(cmd[0], cmd[:1]...) // #nosec G204
	} else {
		c = exec.Command("sudo", cmd...)
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Str("domain", domain).Msg("could not reboot domain")
		return err
	}

	return nil
}

func (l *LibvirtCmd) StartDomain(domain string) error {
	cmd := []string{
		"virsh", "start", domain,
	}

	var c *exec.Cmd
	if l.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Str("domain", domain).Msg("could not start domain")
		return err
	}
	return nil
}

func (l *LibvirtCmd) ShutdownDomain(domain string) error {
	cmd := []string{
		"virsh", "shutdown", domain,
	}

	var c *exec.Cmd
	if l.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Str("domain", domain).Msg("could not shutdown domain")
		return err
	}
	return nil
}
