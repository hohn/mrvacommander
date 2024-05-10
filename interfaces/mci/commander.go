package mci

type Commander interface {
}

type State struct {
	Commander Commander
	Logger    Logger
	Queue     Queue
	Storage   Storage
	Runner    Runner
}
