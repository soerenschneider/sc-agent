package state

import "github.com/rs/zerolog/log"

type RebootNeeded struct {
	stateful Agent
}

func (s *RebootNeeded) Name() StateName {
	return RebootStateName
}

func (s *RebootNeeded) Success() {
	newState := NewUncertainState(s.stateful)
	s.stateful.SetState(newState)
}

func (s *RebootNeeded) Failure() {
}

func (s *RebootNeeded) Error(err error) {
	log.Error().Err(err).Msgf("'%s' encountered error", s.stateful.CheckerNiceName())
	s.stateful.SetState(&ErrorState{stateful: s.stateful})
}
