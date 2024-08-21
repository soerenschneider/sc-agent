package ports

type K0s interface {
	Start() error
	Stop() error
}
