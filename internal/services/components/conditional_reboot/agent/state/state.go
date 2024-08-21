package state

import (
	"context"
	"errors"
	"strings"
	"time"
)

type StateName string

const (
	InitialStateName   StateName = "initial"
	OkStateName        StateName = "ok"
	RebootStateName    StateName = "reboot"
	UncertainStateName StateName = "uncertain"
	ErrorStateName     StateName = "error"
	badState           StateName = "internal error"
)

func FromString(name string) (StateName, error) {
	switch strings.ToLower(name) {
	case string(OkStateName):
		return OkStateName, nil
	case string(ErrorStateName):
		return ErrorStateName, nil
	case string(UncertainStateName):
		return UncertainStateName, nil
	case string(RebootStateName):
		return RebootStateName, nil
	case string(InitialStateName):
		return InitialStateName, nil
	default:
		return badState, errors.New("unknown")
	}
}

type State interface {
	Success()
	Failure()
	Error(err error)
	Name() StateName
}

type Agent interface {
	GetState() State
	SetState(state State)
	StreakUntilOkState() int
	StreakUntilRebootState() int
	GetStateDuration() time.Duration
	Run(ctx context.Context, req chan Agent) error
	CheckerNiceName() string
}
