package sink

import (
	"github.com/soerenschneider/sc-agent/internal/events"
)

type KafkaEventSink struct {
}

func (e *KafkaEventSink) Accept(event events.Event) error {
	return nil
}
