package state_evaluator

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/agent/state"
)

const StateCheckerOrName = "or"

type StateCheckerOr struct {
	wants map[state.StateName]time.Duration
}

func NewStateCheckerOr(args map[string]string) (*StateCheckerOr, error) {
	parsed, err := parseArgsMap(args)
	if err != nil {
		return nil, err
	}

	return &StateCheckerOr{wants: parsed}, nil
}

func (r *StateCheckerOr) ShouldReboot(group Group) bool {
	for _, agent := range group.Agents() {
		if r.CheckAgent(agent) {
			return true
		}
	}

	return false
}

func (r *StateCheckerOr) CheckAgent(agent state.Agent) bool {
	currentState := agent.GetState().Name()
	for wantedType, wantedFor := range r.wants {
		log.Debug().Str("component", "reboot-manager").Str("checker", StateCheckerOrName).Str("agent", agent.CheckerNiceName()).Msgf("wanted=%s, wantedFor=%v", wantedType, wantedFor)
		if currentState == wantedType {
			log.Debug().Str("component", "reboot-manager").Str("checker", StateCheckerOrName).Str("agent", agent.CheckerNiceName()).Msgf("currentState=%s, duration=%v", currentState, agent.GetStateDuration())
			if agent.GetStateDuration() >= wantedFor {
				return true
			}
		}
	}

	return false
}
