package group

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent/state"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/group/state_evaluator"
)

const tickerInterval = 1 * time.Minute

type Group struct {
	agents         []state.Agent
	stateEvaluator state_evaluator.StateEvaluator
	rebootRequests chan *Group
	name           string
}

func NewGroup(name string, agents []state.Agent, stateEvaluator state_evaluator.StateEvaluator, rebootRequests chan *Group) (*Group, error) {
	if len(name) == 0 {
		return nil, errors.New("could not build group: empty name provided")
	}

	if len(agents) == 0 {
		return nil, errors.New("could not build group: empty agents provided")
	}

	if rebootRequests == nil {
		return nil, errors.New("could not build group: nil channel provided")
	}

	return &Group{
		name:           name,
		agents:         agents,
		stateEvaluator: stateEvaluator,
		rebootRequests: rebootRequests,
	}, nil
}

func (g *Group) GetName() string {
	return g.name
}

func (g *Group) Agents() []state.Agent {
	return g.agents
}

func (g *Group) Start(ctx context.Context) {
	agentUpdates := make(chan state.Agent, len(g.agents))

	for _, agent := range g.agents {
		go func(a state.Agent) {
			if err := a.Run(ctx, agentUpdates); err != nil {
				log.Fatal().Str("component", "conditional-reboot").Err(err).Msgf("could start agent %s", a.CheckerNiceName())
			}
		}(agent)
	}

	ticker := time.NewTicker(tickerInterval)

	go func() {
		for {
			select {
			case agent := <-agentUpdates:
				log.Info().Str("component", "conditional-reboot").Msgf("Received update from agent %s", agent.CheckerNiceName())
				if g.stateEvaluator.ShouldReboot(g) {
					log.Debug().Msgf("Reboot checker returned true")
					g.rebootRequests <- g
				}
				log.Debug().Str("component", "conditional-reboot").Msgf("Reboot checker returned false")
			case <-ticker.C:
				if g.stateEvaluator.ShouldReboot(g) {
					log.Debug().Str("component", "conditional-reboot").Msgf("Reboot checker ticker returned true")
					g.rebootRequests <- g
				}
				log.Debug().Str("component", "conditional-reboot").Msgf("Reboot checker ticker returned false")
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
