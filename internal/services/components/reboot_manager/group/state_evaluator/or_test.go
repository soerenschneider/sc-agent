package state_evaluator

import (
	"context"
	"testing"
	"time"

	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/agent/state"
)

type agent struct {
	duration time.Duration
	state    state.State
}

func (a *agent) GetState() state.State {
	return a.state
}

func (a *agent) SetState(state state.State) {
}

func (a *agent) GetStateDuration() time.Duration {
	return a.duration
}

func (a *agent) StreakUntilOkState() int {
	return 1
}

func (a *agent) StreakUntilRebootState() int {
	return 1
}

func (a *agent) Run(ctx context.Context, req chan state.Agent) error {
	return nil
}

func (a *agent) CheckerNiceName() string {
	return ""
}

type args struct {
	agents []state.Agent
}

func (a *args) Agents() []state.Agent {
	return a.agents
}

func TestStateCheckerOr_ShouldReboot(t *testing.T) {

	tests := []struct {
		name  string
		wants map[state.StateName]time.Duration
		args  *args
		want  bool
	}{
		{
			name: "one agent matches constraint, the other doesn't",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(5) * time.Hour,
						state:    &state.NoRebootNeeded{},
					},
					&agent{
						duration: time.Duration(5) * time.Second,
						state:    &state.NoRebootNeeded{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.OkStateName: time.Duration(4) * time.Hour,
			},
			want: true,
		},
		{
			name: "no agent matches constraint",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(5) * time.Minute,
						state:    &state.NoRebootNeeded{},
					},
					&agent{
						duration: time.Duration(5) * time.Second,
						state:    &state.NoRebootNeeded{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.OkStateName: time.Duration(1) * time.Hour,
			},
			want: false,
		},
		{
			name: "both agents match constraint",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(5) * time.Minute,
						state:    &state.NoRebootNeeded{},
					},
					&agent{
						duration: time.Duration(5) * time.Second,
						state:    &state.NoRebootNeeded{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.OkStateName: time.Duration(5) * time.Second,
			},
			want: true,
		},
		{
			name: "no state matches",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(5) * time.Minute,
						state:    &state.ErrorState{},
					},
					&agent{
						duration: time.Duration(5) * time.Second,
						state:    &state.UncertainState{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.OkStateName: time.Duration(5) * time.Second,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StateCheckerOr{
				wants: tt.wants,
			}
			if got := r.ShouldReboot(tt.args); got != tt.want {
				t.Errorf("ShouldReboot() = %v, want %v", got, tt.want)
			}
		})
	}
}
