package state

import "github.com/rs/zerolog/log"

type ErrorState struct {
	stateful    Agent
	errorStreak int
}

func (s *ErrorState) Name() StateName {
	return ErrorStateName
}

func (s *ErrorState) Success() {
	newState := NewUncertainState(s.stateful)
	s.stateful.SetState(newState)
}

func (s *ErrorState) Failure() {
	s.stateful.SetState(&RebootNeeded{stateful: s.stateful})
}

func (s *ErrorState) Error(err error) {
	// try not to flood the logs...
	s.errorStreak++
	if s.errorStreak > 3 {
		log.Debug().Err(err).Msgf("'%s' encountered error", s.stateful.CheckerNiceName())
	} else {
		log.Error().Err(err).Msgf("'%s' encountered error", s.stateful.CheckerNiceName())
	}
}
