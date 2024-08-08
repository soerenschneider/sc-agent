package composites

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/soerenschneider/sc-agent/internal/events"
)

type BackoffComposite struct {
	sink        events.EventSink
	backoffImpl backoff.BackOff
}

func NewRetrier(sink events.EventSink) (*BackoffComposite, error) {
	opts := []backoff.ExponentialBackOffOpts{
		backoff.WithMaxElapsedTime(5 * time.Minute),
	}

	ret := &BackoffComposite{
		sink:        sink,
		backoffImpl: backoff.NewExponentialBackOff(opts...),
	}

	return ret, nil
}

func (e *BackoffComposite) Accept(ctx context.Context, event events.Event) error {
	op := func() error {
		return e.sink.Accept(ctx, event)
	}
	return backoff.Retry(op, e.backoffImpl)
}
