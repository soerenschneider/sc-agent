package deps

import (
	"strings"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/agent/state"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/group"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/group/state_evaluator"
)

func BuildGroup(groupUpdates chan *group.Group, conf *config.GroupConf) (*group.Group, error) {
	agents, err := BuildAgents(conf)
	if err != nil {
		return nil, err
	}

	evaluator, err := BuildStateEvaluator(conf)
	if err != nil {
		return nil, err
	}

	group, err := group.NewGroup(conf.Name, agents, evaluator, groupUpdates)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func BuildAgents(conf *config.GroupConf) ([]state.Agent, error) {
	var agents []state.Agent
	for _, agentConf := range conf.Agents {
		agentConf := agentConf
		agent, err := BuildAgent(&agentConf)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func BuildStateEvaluator(conf *config.GroupConf) (state_evaluator.StateEvaluator, error) {
	switch strings.ToLower(conf.StateEvaluatorName) {
	case state_evaluator.StateCheckerAndName:
		return state_evaluator.NewStateCheckerAnd(conf.StateEvaluatorArgs)
	}

	return state_evaluator.NewStateCheckerOr(conf.StateEvaluatorArgs)
}
