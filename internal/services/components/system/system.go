package system

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/domain/system"
	"go.uber.org/multierr"
)

const serviceName = "machine"

type MachineCmd struct {
	useSudo bool
}

func New(conf config.PowerStatus) (*MachineCmd, error) {
	return &MachineCmd{useSudo: conf.UseSudo}, nil
}

func (m *MachineCmd) SetCpuGovernor(governor system.Governor) error {
	var errs error
	for cpuIndex := 0; cpuIndex < runtime.NumCPU(); cpuIndex++ {
		err := setCPUGovernor(cpuIndex, governor)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	if errs != nil {
		return fmt.Errorf("could not set cpu frequency: %w", errs)
	}
	return nil
}

func (m *MachineCmd) GetCurrentCpuGovernor() (string, error) {
	governorPath := "/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"

	governor, err := os.ReadFile(governorPath)
	if err != nil {
		return "", fmt.Errorf("failed to read governor for CPU0: %v", err)
	}

	return strings.TrimSpace(string(governor)), nil
}

func setCPUGovernor(cpuIndex int, governor system.Governor) error {
	cpuPath := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_governor", cpuIndex)

	if _, err := os.Stat(cpuPath); os.IsNotExist(err) {
		return fmt.Errorf("CPU%d or cpufreq not supported on this system", cpuIndex)
	}

	//nolint G306
	return os.WriteFile(cpuPath, []byte(governor+"\n"), 0644)
}

func (m *MachineCmd) Shutdown() error {
	cmd := []string{"systemctl", "poweroff"}

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
