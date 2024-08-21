package state_evaluator

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent/state"
)

const StateCheckerAndName = "and"

type StateCheckerAnd struct {
	wants map[state.StateName]time.Duration
}

func NewStateCheckerAnd(args map[string]string) (*StateCheckerAnd, error) {
	parsed, err := parseArgsMap(args)
	if err != nil {
		return nil, err
	}
	return &StateCheckerAnd{wants: parsed}, nil
}

func (r *StateCheckerAnd) ShouldReboot(group Group) bool {
	for _, agent := range group.Agents() {
		if !r.CheckAgent(agent) {
			return false
		}
	}

	return true
}

func (r *StateCheckerAnd) CheckAgent(agent state.Agent) bool {
	currentState := agent.GetState().Name()
	for wantedType, wantedFor := range r.wants {
		log.Debug().Str("checker", StateCheckerAndName).Str("agent", agent.CheckerNiceName()).Msgf("wanted=%s, wantedFor=%v", wantedType, wantedFor)
		if currentState == wantedType {
			log.Debug().Str("checker", StateCheckerAndName).Str("agent", agent.CheckerNiceName()).Msgf("currentState=%s, duration=%v", currentState, agent.GetStateDuration())
			if agent.GetStateDuration() >= wantedFor {
				return true
			}
		}
	}

	return false
}
