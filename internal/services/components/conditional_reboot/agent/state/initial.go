package state

import "github.com/rs/zerolog/log"

type InitialState struct {
	stateful Agent
}

func NewInitialState(agent Agent) (*InitialState, error) {
	return &InitialState{stateful: agent}, nil
}

func (s *InitialState) Name() StateName {
	return InitialStateName
}

func (s *InitialState) Failure() {
	var newState State
	if s.stateful.StreakUntilRebootState() > 1 {
		newState = NewUncertainState(s.stateful)
	} else {
		newState = &RebootNeeded{stateful: s.stateful}
	}

	s.stateful.SetState(newState)
}

func (s *InitialState) Success() {
	var newState State
	if s.stateful.StreakUntilOkState() > 1 {
		newState = NewUncertainState(s.stateful)
	} else {
		newState = &NoRebootNeeded{stateful: s.stateful}
	}

	s.stateful.SetState(newState)
}

func (s *InitialState) Error(err error) {
	log.Error().Err(err).Msgf("'%s' encountered error", s.stateful.CheckerNiceName())
	s.stateful.SetState(&ErrorState{stateful: s.stateful})
}
