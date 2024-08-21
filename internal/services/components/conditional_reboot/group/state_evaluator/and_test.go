package state_evaluator

import (
	"testing"
	"time"

	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent/state"
)

func TestStateCheckerAnd_ShouldReboot(t *testing.T) {
	tests := []struct {
		name  string
		wants map[state.StateName]time.Duration
		args  *args
		want  bool
	}{
		{
			name: "both agents match constraint",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(5) * time.Minute,
						state:    &state.ErrorState{},
					},
					&agent{
						duration: time.Duration(5) * time.Second,
						state:    &state.ErrorState{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.ErrorStateName: time.Duration(5) * time.Second,
			},
			want: true,
		},
		{
			name: "no agents match constraint",
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
			name: "one agent matches constraint",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(5) * time.Minute,
						state:    &state.RebootNeeded{},
					},
					&agent{
						duration: time.Duration(5) * time.Second,
						state:    &state.RebootNeeded{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.RebootStateName: time.Duration(1) * time.Minute,
			},
			want: false,
		},
		{
			name: "two states, both match",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(2) * time.Minute,
						state:    &state.RebootNeeded{},
					},
					&agent{
						duration: time.Duration(5) * time.Minute,
						state:    &state.NoRebootNeeded{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.RebootStateName: time.Duration(1) * time.Minute,
				state.OkStateName:     time.Duration(5) * time.Minute,
			},
			want: true,
		},
		{
			name: "two states, one matches",
			args: &args{
				agents: []state.Agent{
					&agent{
						duration: time.Duration(2) * time.Minute,
						state:    &state.RebootNeeded{},
					},
					&agent{
						duration: time.Duration(1) * time.Minute,
						state:    &state.NoRebootNeeded{},
					},
				},
			},
			wants: map[state.StateName]time.Duration{
				state.RebootStateName: time.Duration(1) * time.Minute,
				state.OkStateName:     time.Duration(5) * time.Minute,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StateCheckerAnd{
				wants: tt.wants,
			}
			if got := r.ShouldReboot(tt.args); got != tt.want {
				t.Errorf("ShouldReboot() = %v, want %v", got, tt.want)
			}
		})
	}
}
