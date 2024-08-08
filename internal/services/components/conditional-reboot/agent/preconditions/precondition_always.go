package preconditions

const AlwaysPreconditionName = "always"

type AlwaysPrecondition struct {
}

func (c *AlwaysPrecondition) PerformCheck() bool {
	return true
}
