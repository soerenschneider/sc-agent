package config

import (
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent/state"
	"gopkg.in/yaml.v3"
)

const (
	defaultStreakUntilOk      = 3
	defaultStreakUntilReboot  = 1
	defaultStateEvaluatorName = "or"
)

type ConditionalRebootConfig struct {
	Enabled     bool        `yaml:"enabled"`
	Groups      []GroupConf `yaml:"groups" validate:"dive,required"`
	JournalFile string      `yaml:"journal_file" validate:"omitempty,filepath"`
}

func (conf *ConditionalRebootConfig) Print() {
	log.Info().Msg("Active config values:")
	for _, group := range conf.Groups {
		log.Info().Msgf("Group '%s', stateEvaluator='%s', stateEvaluatorArgs=%v", group.Name, group.StateEvaluatorName, group.StateEvaluatorArgs)
		for _, agent := range group.Agents {
			log.Info().Msgf("--> Agent '%s', checkerArgs=%v, precondition='%s', preconditionArgs=%v, streakUntilRecovered=%d, streakUntilUnhealthy=%d", agent.CheckerName, agent.CheckerArgs, agent.PreconditionName, agent.PreconditionArgs, agent.StreakUntilOk, agent.StreakUntilReboot)
		}
	}
}

type GroupConf struct {
	Agents             []AgentConf       `yaml:"agents" validate:"dive"`
	Name               string            `yaml:"name" validate:"required"`
	StateEvaluatorName string            `yaml:"state_evaluator_name"`
	StateEvaluatorArgs map[string]string `yaml:"state_evaluator_args" validate:"required"`
}

func (conf *GroupConf) UnmarshalYAML(node *yaml.Node) error {
	type Alias GroupConf // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{
		StateEvaluatorName: defaultStateEvaluatorName,
		StateEvaluatorArgs: map[string]string{
			string(state.RebootStateName): "0s",
		},
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = GroupConf(*tmp)
	return nil
}

type AgentConf struct {
	CheckInterval     string `yaml:"check_interval" validate:"required"`
	StreakUntilOk     int    `yaml:"streak_until_ok" validate:"required,gte=1"`
	StreakUntilReboot int    `yaml:"streak_until_reboot" validate:"gte=1"`

	CheckerName string         `yaml:"checker_name" validate:"required"`
	CheckerArgs map[string]any `yaml:"checker_args"`

	PreconditionName string         `yaml:"precondition_name"`
	PreconditionArgs map[string]any `yaml:"precondition_args"`
}

func (conf *AgentConf) UnmarshalYAML(node *yaml.Node) error {
	type Alias AgentConf // Create an alias to avoid recursion during unmarshalling

	// Define a temporary struct with default values
	tmp := &Alias{
		StreakUntilReboot: defaultStreakUntilReboot,
		StreakUntilOk:     defaultStreakUntilOk,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = AgentConf(*tmp)
	return nil
}
