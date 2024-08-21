package preconditions

// Precondition defines a condition that has to be met before a Checker is even executed.
type Precondition interface {
	// PerformCheck returns true if the Agent should continue with performing its configured Checker
	PerformCheck() bool
}
