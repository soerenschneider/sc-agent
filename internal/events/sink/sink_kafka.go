package sink

import (
	"encoding/json"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/soerenschneider/sc-agent/internal/events"
)

type KafkaEventSink struct {
}

func (e *KafkaEventSink) Accept(event events.Event) error {
	marshalled, err := json.Marshal(event)
	if err != nil {
		return backoff.Permanent(err)
	}

	fmt.Println(marshalled)

	return nil
}
