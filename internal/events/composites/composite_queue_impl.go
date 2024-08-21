package composites

import (
	"errors"

	"github.com/adrianbrad/queue"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

var ErrNoElementsAvailable = errors.New("no elements available in the queue")

type CircularQueue[T comparable] struct {
	queue *queue.Circular[T]
}

func NewQueue[T comparable](size int) (*CircularQueue[T], error) {
	if size < 100 || size > 100_000 {
		return nil, errors.New("size must be >= 100 and <= 100.000")
	}

	metrics.QueueCapacity.Set(float64(size))
	return &CircularQueue[T]{
		queue: queue.NewCircular([]T{}, size),
	}, nil
}

func (q *CircularQueue[T]) Offer(item T) error {
	log.Debug().Msg("Inserting item to queue")
	err := q.queue.Offer(item)
	metrics.QueueSize.Set(float64(q.queue.Size()))
	return err
}

func (q *CircularQueue[T]) Get() (T, error) {
	metrics.QueueSize.Set(float64(q.queue.Size()))
	log.Debug().Msg("Removing head from from queue")
	item, err := q.queue.Get()
	return item, err
}

func (q *CircularQueue[T]) IsEmpty() bool {
	return q.queue.IsEmpty()
}
