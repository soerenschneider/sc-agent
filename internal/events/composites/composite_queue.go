package composites

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/events"
)

type Queue interface {
	Offer(item events.Event) error
	Get() (events.Event, error)
	IsEmpty() bool
}

type QueueComposite struct {
	sink  events.EventSink
	queue Queue
}

func (e *QueueComposite) Work(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)

	stop := false
	for !stop {
		select {
		case <-ctx.Done():
			stop = true
		case <-ticker.C:
			if err := e.workQueue(); err != nil {
				log.Error().Err(err).Msg("could not iterate")
			}
		}
	}
}

func (e *QueueComposite) Accept(ctx context.Context, event events.Event) error {
	err := e.sink.Accept(ctx, event)
	if err != nil {
		return e.queue.Offer(event)
	}

	return nil
}

func (e *QueueComposite) workQueue() error {
	read, err := e.queue.Get()
	if err != nil && !errors.Is(err, ErrNoElementsAvailable) {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := e.sink.Accept(ctx, read); err != nil {
		return e.queue.Offer(read)
	}

	return nil
}
