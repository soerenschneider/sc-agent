package events

import "context"

type Event struct {
	Id     string
	Source string
	Type   string
	Data   string
}

type EventSink interface {
	Accept(ctx context.Context, event Event) error
}
