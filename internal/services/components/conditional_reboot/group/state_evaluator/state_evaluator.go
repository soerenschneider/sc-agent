package state_evaluator

import (
	"errors"
	"fmt"
	"time"

	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent/state"
)

// StateEvaluator checks a Group and decides whether its state (and the time the Agent resides in the current state)
// justifies a reboot.
type StateEvaluator interface {
	ShouldReboot(group Group) bool
}

type Group interface {
	Agents() []state.Agent
}

func parseArgsMap(args map[string]string) (map[state.StateName]time.Duration, error) {
	if len(args) == 0 {
		return nil, errors.New("empty args provided")
	}

	ret := map[state.StateName]time.Duration{}
	for name, duration := range args {
		stateName, err := state.FromString(name)
		if err != nil {
			return nil, fmt.Errorf("not a valid StateName '%s'", name)
		}

		parsedDuration, err := time.ParseDuration(duration)
		if err != nil {
			return nil, fmt.Errorf("could not build '%s' state checker for state '%s': could not parse duration: %w", StateCheckerAndName, name, err)
		}

		if parsedDuration < 0 {
			return nil, fmt.Errorf("could not build '%s' state checker for state '%s': duration may not be < 0 (you supplied '%s')", StateCheckerAndName, name, duration)
		}

		if parsedDuration > 24*time.Hour {
			return nil, fmt.Errorf("could not build '%s' state checker for state '%s': duration may not be > 24h (you supplied '%s')", StateCheckerAndName, name, duration)
		}

		ret[stateName] = parsedDuration
	}

	return ret, nil
}
