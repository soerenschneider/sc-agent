package state

import "github.com/rs/zerolog/log"

type NoRebootNeeded struct {
	stateful Agent
}

func (s *NoRebootNeeded) Name() StateName {
	return OkStateName
}

func (s *NoRebootNeeded) Success() {
}

func (s *NoRebootNeeded) Failure() {
	if s.stateful.StreakUntilRebootState() > 1 {
		s.stateful.SetState(NewUncertainState(s.stateful))
	} else {
		s.stateful.SetState(&RebootNeeded{stateful: s.stateful})
	}
}

func (s *NoRebootNeeded) Error(err error) {
	log.Error().Err(err).Msgf("'%s' encountered error", s.stateful.CheckerNiceName())
	s.stateful.SetState(&ErrorState{stateful: s.stateful})
}
