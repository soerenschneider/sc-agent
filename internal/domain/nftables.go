package domain

var (
	ErrNftablesNoSuchSet      = newCategorizedError(ErrNotFound, "set not found")
	ErrNftablesInvalidElement = newCategorizedError(ErrInvalidInput, "invalid element")
	ErrNftablesInvalidTimeout = newCategorizedError(ErrInvalidInput, "invalid timeout")
)
