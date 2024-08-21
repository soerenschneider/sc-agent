package systemd

import (
	"bytes"
	"cmp"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

const (
	serviceName  = "systemd"
	defaultLines = 100
)

type SystemdCmd struct {
	useSudo bool

	unitsAllowlist []string
	unitsDenylist  []string
}

func New(conf config.Services) (*SystemdCmd, error) {
	ret := &SystemdCmd{
		useSudo:        conf.UseSudo,
		unitsAllowlist: conf.UnitsAllowlist,
		unitsDenylist:  conf.UnitsDenylist,
	}

	return ret, nil
}

func (s *SystemdCmd) Restart(unit string) error {
	if len(s.unitsDenylist) > 0 && slices.Contains(s.unitsDenylist, unit) {
		return domain.ErrPermissionDenied
	}

	if len(s.unitsAllowlist) > 0 && !slices.Contains(s.unitsAllowlist, unit) {
		return domain.ErrPermissionDenied
	}

	cmd := []string{"systemctl", "restart", unit}

	var c *exec.Cmd
	if s.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Str("unit", unit).Msg("could not restart systemd unit")
		return err
	}
	return nil
}

func (s *SystemdCmd) Logs(req ports.SystemdLogsRequest) ([]string, error) {
	numberOfLines := cmp.Or(req.Lines, defaultLines)
	cmd := []string{"journalctl", "-n", strconv.Itoa(numberOfLines), "--unit", req.Unit}

	s.useSudo = true
	var c *exec.Cmd
	if s.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	stdout := &bytes.Buffer{}
	c.Stdout = stdout

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("service", serviceName).Str("unit", req.Unit).Msg("could not gather systemd logs")
		return nil, err
	}

	output := stdout.String()
	if strings.TrimSpace(output) == "-- No entries --" {
		return nil, domain.ErrServicesNoSuchUnit
	}

	return strings.Split(stdout.String(), "\n"), nil
}
