package composites

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/soerenschneider/sc-agent/internal/events"
)

type BackoffComposite struct {
	sink events.EventSink
}

func NewRetrier(sink events.EventSink) (*BackoffComposite, error) {
	ret := &BackoffComposite{
		sink: sink,
	}

	return ret, nil
}

func (e *BackoffComposite) Accept(ctx context.Context, event cloudevents.Event) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxElapsedTime = 30 * time.Second

	op := func() error {
		return e.sink.Accept(ctx, event)
	}

	return backoff.Retry(op, backoff.WithContext(expBackoff, ctx))
}
