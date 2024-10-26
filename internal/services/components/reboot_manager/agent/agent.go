package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/agent/state"

	"sync"
	"time"
)

type Checker interface {
	IsHealthy(ctx context.Context) (bool, error)
	Name() string
}

// Precondition defines a condition that has to be met before a Checker is even executed.
type Precondition interface {
	// PerformCheck returns true if the Agent should continue with performing its configured Checker
	PerformCheck() bool
}

type StatefulAgent struct {
	checker       Checker
	precondition  Precondition
	checkInterval time.Duration

	//durationUntilRecovered specifies the duration that the state "recovering" needs to be in to become "healthy" again.
	updateChannel chan state.Agent

	streakUntilOk           int
	streakUntilRebootNeeded int

	state           state.State
	lastStateChange time.Time
	mutex           sync.RWMutex
}

func NewAgent(checker Checker, precondition Precondition, conf *config.AgentConf) (*StatefulAgent, error) {
	if checker == nil {
		return nil, errors.New("could not build agent: empty checker supplied")
	}

	if precondition == nil {
		return nil, errors.New("could not build agent: empty precondition supplied")
	}

	if conf == nil {
		return nil, errors.New("empty agent conf provided")
	}

	parsedCheckInterval, err := time.ParseDuration(conf.CheckInterval)
	if err != nil {
		return nil, fmt.Errorf("can not parse 'checkInterval' duration string '%s'", conf.CheckInterval)
	}

	if parsedCheckInterval < time.Duration(5)*time.Second {
		return nil, fmt.Errorf("'checkInterval' may not be < 5s")
	}

	if parsedCheckInterval > time.Duration(1)*time.Hour {
		return nil, fmt.Errorf("'checkInterval' may not be > 1h")
	}

	agent := &StatefulAgent{
		checker:                 checker,
		precondition:            precondition,
		checkInterval:           parsedCheckInterval,
		streakUntilRebootNeeded: conf.StreakUntilReboot,
		streakUntilOk:           conf.StreakUntilOk,
		lastStateChange:         time.Now(),
	}

	agent.state, err = state.NewInitialState(agent)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (a *StatefulAgent) Run(ctx context.Context, stateUpdateChannel chan state.Agent) error {
	if stateUpdateChannel == nil {
		return errors.New("empty channel provided")
	}
	a.updateChannel = stateUpdateChannel

	a.performCheck(ctx)
	ticker := time.NewTicker(a.checkInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil
		case <-ticker.C:
			a.performCheck(ctx)
		}
	}
}

func (a *StatefulAgent) performCheck(ctx context.Context) {
	log.Debug().Str("component", "reboot-manager").Msgf("performCheck() %s", a.CheckerNiceName())
	if !a.precondition.PerformCheck() {
		log.Debug().Str("component", "reboot-manager").Msgf("Precondition not met, not invoking checker %s", a.CheckerNiceName())
		return
	}

	metrics.CheckerLastCheck.WithLabelValues(a.checker.Name()).SetToCurrentTime()

	log.Debug().Str("component", "reboot-manager").Msgf("IsHealthy() %s", a.CheckerNiceName())
	isHealthy, err := a.checker.IsHealthy(ctx)
	if err != nil {
		log.Warn().Str("component", "reboot-manager").Str("checker", a.CheckerNiceName()).Msg("can not determine healthiness")
		a.state.Error(err)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "err").Set(1)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "healthy").Set(0)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "unhealthy").Set(0)
		return
	}

	if isHealthy {
		log.Debug().Str("component", "reboot-manager").Str("checker", a.CheckerNiceName()).Msg("is healthy")
		a.state.Success()
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "err").Set(0)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "healthy").Set(1)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "unhealthy").Set(0)
	} else {
		log.Debug().Str("component", "reboot-manager").Str("checker", a.CheckerNiceName()).Msg("is UNHEALTHY!")
		a.state.Failure()
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "err").Set(0)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "healthy").Set(0)
		metrics.CheckerState.WithLabelValues(a.checker.Name(), "unhealthy").Set(1)
	}
}

func (a *StatefulAgent) SetState(newState state.State) {
	log.Info().Str("component", "reboot-manager").Msgf("Updating state for checker '%s' from '%s' -> '%s'", a.checker.Name(), a.state.Name(), newState.Name())

	metrics.AgentState.WithLabelValues(string(newState.Name()), a.CheckerNiceName()).Set(1)
	metrics.AgentState.WithLabelValues(string(a.state.Name()), a.CheckerNiceName()).Set(0)
	metrics.LastStateChange.WithLabelValues(string(a.state.Name()), a.CheckerNiceName()).SetToCurrentTime()

	log.Debug().Str("component", "reboot-manager").Str("checker", a.CheckerNiceName()).Msgf("SetState(%s) acquire lock", newState.Name())
	a.mutex.Lock()
	defer a.mutex.Unlock()
	log.Debug().Str("component", "reboot-manager").Str("checker", a.CheckerNiceName()).Msgf("SetState(%s) success", newState.Name())

	a.lastStateChange = time.Now()
	a.state = newState
	log.Debug().Str("component", "reboot-manager").Msgf("Updating channel %s", a.checker.Name())
	a.updateChannel <- a
	log.Debug().Str("component", "reboot-manager").Msgf("Updated channel %s", a.checker.Name())
}

func (a *StatefulAgent) String() string {
	return fmt.Sprintf("%s checker=%s, checkInterval=%s, streakUntilOk=%d, streakUntilUnhealhty=%d", a.CheckerNiceName(), a.checker.Name(), a.checkInterval, a.streakUntilOk, a.streakUntilRebootNeeded)
}

func (a *StatefulAgent) CheckerNiceName() string {
	return a.checker.Name()
}

func (a *StatefulAgent) StreakUntilOkState() int {
	return a.streakUntilOk
}

func (a *StatefulAgent) StreakUntilRebootState() int {
	return a.streakUntilRebootNeeded
}

func (a *StatefulAgent) GetState() state.State {
	log.Debug().Str("component", "reboot-manager").Msgf("GetState() acquire lock (%s)", a.CheckerNiceName())
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	log.Debug().Str("component", "reboot-manager").Msgf("GetState() lock success (%s)", a.CheckerNiceName())

	return a.state
}

func (a *StatefulAgent) GetStateDuration() time.Duration {
	return time.Since(a.lastStateChange)
}

func (a *StatefulAgent) Failure() {
	a.state.Failure()
}

func (a *StatefulAgent) Success() {
	a.state.Success()
}

func (a *StatefulAgent) Error(err error) {
	a.state.Error(err)
}
