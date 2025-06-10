package main

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/events"
	"github.com/soerenschneider/sc-agent/internal/events/composites"
	"github.com/soerenschneider/sc-agent/internal/events/sink"
)

func buildEventSink(conf config.Config, services *ports.Components) (events.EventSink, error) {
	if conf.Nats == nil || !conf.Nats.Enabled {
		return nil, nil
	}

	nats, err := sink.NewNatsNotification(conf.Nats.Url, conf.Nats.Subject)
	if err != nil {
		return nil, fmt.Errorf("could not build nats event sink: %w", err)
	}

	backoffComposite, err := composites.NewRetrier(nats)
	if err != nil {
		return nil, fmt.Errorf("could not build retrier composite: %w", err)
	}

	queueComposite, err := composites.NewQueueEventSink(backoffComposite)
	if err != nil {
		return nil, fmt.Errorf("could not build queue composite: %w", err)
	}

	return queueComposite, nil
}
