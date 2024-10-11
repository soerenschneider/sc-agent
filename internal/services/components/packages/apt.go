package packages

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const (
	aptSubcomponent         = "apt"
	packageManagerComponent = "package-manager"
)

type AptPackageManager struct {
	useSudo      bool
	upgradeMutex sync.Mutex
}

func NewAptPackageManager() (*AptPackageManager, error) {
	return &AptPackageManager{}, nil
}

func (m *AptPackageManager) ListInstalled() ([]domain.PackageInfo, error) {
	cmd := []string{"apt", "list", "--installed"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	stdout := &bytes.Buffer{}
	c.Stdout = stdout

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("component", packageManagerComponent).Str("subcomponent", aptSubcomponent).Msg("could not get list of installed packages")
		return nil, err
	}

	installed := parseAptListInstalledOutput(stdout.String())
	metrics.PackagesInstalled.Set(float64(len(installed)))
	return installed, nil
}

func (m *AptPackageManager) Upgrade() error {
	cmd := []string{"apt", "-y", "upgrade"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.upgradeMutex.Lock()
		defer m.upgradeMutex.Unlock()
		log.Info().Str("component", packageManagerComponent).Str("subcomponent", aptSubcomponent).Msg("start system upgrade")
		if err := c.Start(); err != nil {
			metrics.DnfErrors.WithLabelValues("Upgrade").Inc()
			log.Error().Str("component", packageManagerComponent).Str("subcomponent", aptSubcomponent).Err(err).Msg("error running upgrade")
		}

		wg.Done()
		if err := c.Wait(); err != nil {
			metrics.DnfErrors.WithLabelValues("Upgrade").Inc()
			log.Error().Str("component", packageManagerComponent).Str("subcomponent", aptSubcomponent).Err(err).Msg("error running upgrade")
		} else {
			log.Info().Str("component", packageManagerComponent).Str("subcomponent", aptSubcomponent).Msg("upgrade successfully applied")
		}
	}()
	wg.Wait()
	return nil
}

func (m *AptPackageManager) CheckUpdate() (domain.CheckUpdateResult, error) {
	metrics.UpdateCheckAvailableTimestamp.SetToCurrentTime()
	cmd := []string{"apt", "list", "--upgradeable", "-qq"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	stdout := &bytes.Buffer{}
	c.Stdout = stdout

	result := domain.CheckUpdateResult{
		UpdatesAvailable: false,
	}
	err := c.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			switch exitError.ExitCode() {
			case 0:
				packages := parseAptListUpgradeableOutput(stdout.String())
				metrics.UpdatesAvailable.Set(float64(len(packages.UpdatablePackages)))
				if packages.UpdatesAvailable {
					metrics.UpdatesAvailableBool.Set(1)
				} else {
					metrics.UpdatesAvailableBool.Set(0)
				}
				return result, err
			default:
				log.Warn().Str("component", packageManagerComponent).Str("subcomponent", aptSubcomponent).Int("exitcode", exitError.ExitCode()).Msg("got unknown exitcode from running dnf check-update")
			}
		}
	}

	metrics.UpdatesAvailableBool.Set(0)
	metrics.UpdatesAvailable.Set(0)
	return result, nil
}

func parseAptListInstalledOutput(output string) []domain.PackageInfo {
	var ret []domain.PackageInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			nameParts := strings.Split(parts[0], "/")
			if len(nameParts) > 0 {
				ret = append(ret, domain.PackageInfo{
					Name:    nameParts[0],
					Version: parts[1],
					Repo:    nameParts[1],
				})
			}
		}
	}

	return ret
}

func parseAptListUpgradeableOutput(output string) domain.CheckUpdateResult {
	var updatablePackages []domain.PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Split(line, "/")
		if len(parts) > 0 {
			name := parts[0]
			updatablePackages = append(updatablePackages, domain.PackageInfo{
				Name: name,
			})
		}
	}

	return domain.CheckUpdateResult{
		UpdatablePackages: updatablePackages,
		UpdatesAvailable:  len(updatablePackages) > 0,
	}
}
