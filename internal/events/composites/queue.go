package composites

import (
	"encoding/json"
	"errors"

	"github.com/adrianbrad/queue"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

type CircularQueue struct {
	queue *queue.Circular[string]
}

func NewQueue(size int) (*CircularQueue, error) {
	if size < 100 {
		return nil, errors.New("size must not be less than 100")
	}

	metrics.EventQueueCapacity.Set(float64(size))
	return &CircularQueue{
		queue: queue.NewCircular[string](nil, size),
	}, nil
}

func (q *CircularQueue) Offer(item cloudevents.Event) error {
	//b, err := msgpack.Marshal(item)
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	err = q.queue.Offer(string(b))
	metrics.EventQueueSize.Set(float64(q.queue.Size()))
	return err
}

func (q *CircularQueue) Get() (cloudevents.Event, error) {
	data, err := q.queue.Get()
	if err != nil {
		return cloudevents.Event{}, err
	}
	metrics.EventQueueSize.Set(float64(q.queue.Size()))
	item := cloudevents.Event{}
	//if err := msgpack.Unmarshal([]byte(data), &item); err != nil {
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return cloudevents.Event{}, err
	}
	return item, nil
}

func (q *CircularQueue) IsEmpty() bool {
	return q.queue.IsEmpty()
}
