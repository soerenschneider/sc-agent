package packages

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const (
	dnfSubcomponent = "dnf"
)

var dnfOutputRegex = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s*$`)

type DnfPackageManager struct {
	useSudo      bool
	upgradeMutex sync.Mutex
}

func NewDnfPackageManager() (*DnfPackageManager, error) {
	return &DnfPackageManager{}, nil
}

func (m *DnfPackageManager) ListInstalled() ([]domain.PackageInfo, error) {
	cmd := []string{"dnf", "--color=never", "list", "--installed"}

	var c *exec.Cmd
	if m.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	stdout := &bytes.Buffer{}
	c.Stdout = stdout

	if err := c.Run(); err != nil {
		log.Error().Err(err).Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Msg("could not get list of installed packages")
		return nil, err
	}

	installed := parseDnfListOutput(stdout.String())
	metrics.PackagesInstalled.Set(float64(len(installed)))
	return installed, nil
}

func (m *DnfPackageManager) Upgrade() error {
	cmd := []string{"dnf", "--color=never", "upgrade", "-y"}

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
		log.Info().Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Msg("start system upgrade")
		if err := c.Start(); err != nil {
			metrics.DnfErrors.WithLabelValues("Upgrade").Inc()
			log.Error().Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Err(err).Msg("error running upgrade")
		}

		wg.Done()
		if err := c.Wait(); err != nil {
			metrics.DnfErrors.WithLabelValues("Upgrade").Inc()
			log.Error().Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Err(err).Msg("error running upgrade")
		} else {
			log.Info().Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Msg("upgrade successfully applied")
		}
	}()
	wg.Wait()
	return nil
}

func (m *DnfPackageManager) CheckUpdate() (domain.CheckUpdateResult, error) {
	metrics.UpdateCheckAvailableTimestamp.SetToCurrentTime()
	cmd := []string{"dnf", "--color=never", "check-update"}

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
			case 1:
				metrics.UpdatesAvailableBool.Set(0)
				metrics.UpdatesAvailable.Set(0)
				metrics.DnfErrors.WithLabelValues("CheckUpdateResult").Inc()
				return result, err
			case 100:
				metrics.UpdatesAvailableBool.Set(1)
				result.UpdatesAvailable = true
				packages, err := parseDnfCheckUpdateOutput(stdout.String())
				if err != nil {
					log.Error().Err(err).Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Msg("could not parse output of 'dnf check-update'")
					return result, nil
				}
				metrics.UpdatesAvailable.Set(float64(len(packages)))
				result.UpdatablePackages = packages
				return result, nil
			default:
				log.Warn().Str("component", packageManagerComponent).Str("subcomponent", dnfSubcomponent).Int("exitcode", exitError.ExitCode()).Msg("got unknown exitcode from running dnf check-update")
			}
		}
	}

	metrics.UpdatesAvailableBool.Set(0)
	metrics.UpdatesAvailable.Set(0)
	return result, nil
}

func parseDnfCheckUpdateOutput(output string) ([]domain.PackageInfo, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var packages []domain.PackageInfo

	for scanner.Scan() {
		line := scanner.Text()

		if matches := dnfOutputRegex.FindStringSubmatch(line); matches != nil {
			if len(matches) == 4 {
				packages = append(packages, domain.PackageInfo{
					Name:    matches[1],
					Version: matches[2],
					Repo:    matches[3],
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return packages, nil
}

func parseDnfListOutput(output string) []domain.PackageInfo {
	var ret []domain.PackageInfo

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			name := fields[0]
			version := fields[1]
			repo := fields[2]
			ret = append(ret, domain.PackageInfo{Name: name, Version: version, Repo: repo})
		}
	}

	return ret
}
