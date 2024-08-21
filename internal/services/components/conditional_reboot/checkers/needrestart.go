package checkers

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

const (
	NeedrestartCheckerName = "needrestart"
	DefaultKstaVal         = 2 // A bug in needrestart _always_ yields '2' on Rocky Linux, even directly after a restart
	DefaultRebootOnSvc     = true
)

var kstaRegex = regexp.MustCompile(`NEEDRESTART-KSTA: (?P<ksta>\d)`)

type Needrestart interface {
	Result(ctx context.Context) (string, error)
}

type NeedrestartCmd struct{}

func (n *NeedrestartCmd) Result(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "sudo", "needrestart", "-b").Output()
	if err != nil {
		return "", fmt.Errorf("could not determine if reboot is needed: %w", err)
	}

	return string(out), nil
}

// NeedrestartChecker uses https://github.com/liske/needrestart to check whether rebooting is needed
type NeedrestartChecker struct {
	rebootOnSvc   bool
	rebootMinKsta int

	rebootNeeded bool
	sync         sync.Mutex
	needrestart  Needrestart
}

type NeedRestartOpts func(checker *NeedrestartChecker) error

func NewNeedrestartChecker(options ...NeedRestartOpts) (*NeedrestartChecker, error) {
	checker := &NeedrestartChecker{
		sync:          sync.Mutex{},
		needrestart:   &NeedrestartCmd{},
		rebootMinKsta: DefaultKstaVal,
		rebootOnSvc:   DefaultRebootOnSvc,
	}

	var errs error
	for _, opt := range options {
		err := opt(checker)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return checker, errs
}

func (n *NeedrestartChecker) Name() string {
	return NeedrestartCheckerName
}

func (n *NeedrestartChecker) IsHealthy(ctx context.Context) (bool, error) {
	n.sync.Lock()
	defer n.sync.Unlock()

	// use cached reply
	if n.rebootNeeded {
		return false, nil
	}

	out, err := n.needrestart.Result(ctx)
	if err != nil {
		return false, err
	}

	kernelUpdate, svcUpdates := n.detectUpdates(out)

	// cache a response in case we need to reboot - we won't recover from a needed reboot until we actually reboot
	if n.rebootOnSvc {
		n.rebootNeeded = kernelUpdate || svcUpdates
	} else {
		n.rebootNeeded = kernelUpdate
	}

	// reboot is needed, report unhealthy status
	if n.rebootNeeded {
		return false, nil
	}
	return true, nil
}

func (n *NeedrestartChecker) detectUpdates(out string) (bool, bool) {
	var kernelUpdate, svcUpdates bool

	// check for updated kernel
	matches := kstaRegex.FindStringSubmatch(out)
	if len(matches) >= 2 {
		val, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Error().Str("checker", "needrestart").Msgf("could not parse 'NEEDRESTART-KSTA': %v", err)
		} else if val > n.rebootMinKsta {
			kernelUpdate = true
			log.Info().Str("checker", "needrestart").Int("KSTA", val).Msg("Kernel updates detected")
		}
	} else {
		log.Warn().Str("checker", "needrestart").Msg("Could not find KSTA information")
	}

	// check for service upgrades
	if strings.Contains(out, "NEEDRESTART-SVC:") {
		svcUpdates = true
		log.Info().Str("checker", "needrestart").Msg("Service updates detected")
	}

	return kernelUpdate, svcUpdates
}
