package reboot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

const graceTime = time.Second * 15

type DefaultRebootImpl struct {
}

func (l *DefaultRebootImpl) Reboot() (err error) {
	defer func(err error) {
		if err == nil {
			// Give some time to system to actually reboot
			time.Sleep(graceTime)
		}
	}(err)

	uid := os.Getuid()
	if uid == 0 {
		err = rebootAsRoot()
	} else {
		err = rebootAsUser()
	}

	return
}

func rebootAsRoot() error {
	var errs error

	log.Info().Msg("Running as root, attempting direct reboot...")
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("reboot via syscall did not work: %w", err))
	} else {
		return nil
	}

	log.Info().Msg("Running as root, attempting reboot via systemctl reboot ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "reboot")
	if err := cmd.Run(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("reboot via 'systemctl reboot' did not work: %w", err))
		return errs
	}

	return nil
}

func rebootAsUser() error {
	log.Info().Msg("Not running as root, trying rebooting the system via 'sudo systemctl reboot'... ")

	var errs error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "reboot")
	if err := cmd.Run(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("reboot system using 'sudo systemctl reboot' did not work: %w", err))
	} else {
		return nil
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "sudo", "reboot")
	if err := cmd.Run(); err != nil {
		errs = multierr.Append(errs, fmt.Errorf("reboot system using 'sudo reboot' did not work: %w", err))
		return errs
	}

	return nil
}
