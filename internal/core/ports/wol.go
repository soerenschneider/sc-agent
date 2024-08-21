package ports

type WakeOnLan interface {
	WakeUp(alias string) error
}
