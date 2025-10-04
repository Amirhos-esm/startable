package startable

type startable interface {
	IsStarted() bool
	Start() error
	Stop() error
}
