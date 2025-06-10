package composites

import (
	"context"
	"fmt"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/events"
)

type QueueComposite struct {
	sink   events.EventSink
	queue  *CircularQueue
	once   sync.Once
	ticker *time.Ticker
}

func NewQueueEventSink(sink events.EventSink) (*QueueComposite, error) {
	queue, err := NewQueue(500)
	if err != nil {
		return nil, fmt.Errorf("could not create queue event sink: %w", err)
	}

	ret := &QueueComposite{
		sink:  sink,
		queue: queue,
	}

	return ret, nil
}

func (e *QueueComposite) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.ticker.Stop()
		case <-e.ticker.C:
			var addAgain []cloudevents.Event
			foundEnd := false
			mutex := &sync.Mutex{}
			wg := &sync.WaitGroup{}
			for !foundEnd {
				if e.queue.IsEmpty() {
					foundEnd = true
					continue
				}
				item, err := e.queue.Get()
				if err != nil {
					foundEnd = true
					continue
				} else {
					wg.Add(1)
					go func() {
						ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
						defer func() {
							cancel()
							wg.Done()
						}()

						if err := e.sink.Accept(ctx, item); err != nil {
							mutex.Lock()
							addAgain = append(addAgain, item)
							mutex.Unlock()
						}
					}()
				}
			}

			wg.Wait()

			for _, item := range addAgain {
				_ = e.queue.Offer(item)
			}
		}
	}
}

func (e *QueueComposite) Accept(ctx context.Context, event cloudevents.Event) error {
	e.once.Do(func() {
		e.ticker = time.NewTicker(60 * time.Second)
		go func() {
			e.work(ctx)
		}()
	})

	if err := e.sink.Accept(ctx, event); err != nil {
		log.Warn().Err(err).Msg("could not dispatch event, adding to queue")
	}
	return e.queue.Offer(event)
}
