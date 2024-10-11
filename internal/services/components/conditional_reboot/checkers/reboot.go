package checkers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

const RebootCheckerName = "reboot"

type RebootCheckerDnf struct {
	useSudo bool
}

func NewRebootCheckerDnf() (*RebootCheckerDnf, error) {
	return &RebootCheckerDnf{}, nil
}

func (r *RebootCheckerDnf) Name() string {
	return fmt.Sprintf("%s-dnf", RebootCheckerName)
}

func (r *RebootCheckerDnf) IsHealthy(ctx context.Context) (bool, error) {
	cmd := []string{"dnf", "--color=never", "needs-restarting", "-r"}

	var c *exec.Cmd
	if r.useSudo {
		c = exec.CommandContext(ctx, "sudo", cmd...)
	} else {
		c = exec.CommandContext(ctx, cmd[0], cmd[1:]...) // #nosec G204
	}

	stdout := &bytes.Buffer{}
	c.Stdout = stdout

	err := c.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			switch exitError.ExitCode() {
			case 1:
				return false, nil
			default:
				log.Warn().Int("exitcode", exitError.ExitCode()).Msg("got unknown exitcode from running dnf needs-restarting")
			}
		}
	}

	return true, nil
}

type RebootCheckerApt struct{}

func NewRebootCheckerApt() (*RebootCheckerApt, error) {
	return &RebootCheckerApt{}, nil
}

func (r *RebootCheckerApt) Name() string {
	return fmt.Sprintf("%s-apt", RebootCheckerName)
}

func (r *RebootCheckerApt) IsHealthy(_ context.Context) (bool, error) {
	if _, err := os.Stat("/var/run/reboot-required"); err == nil {
		return false, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return true, nil
	} else {
		return false, err
	}
}
