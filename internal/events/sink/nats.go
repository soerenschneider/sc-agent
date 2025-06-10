package sink

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

type Nats struct {
	url     string
	subject string
	js      jetstream.JetStream

	mutex sync.Mutex
}

func NewNatsNotification(url, subject string) (*Nats, error) {
	ret := &Nats{
		url:     url,
		subject: subject,
	}

	return ret, ret.Connect()
}

func (n *Nats) Connect() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	nc, err := nats.Connect(n.url)
	if err != nil {
		return err
	}

	metrics.NatsConnectionStatus.WithLabelValues("connected").Set(1)
	metrics.NatsConnectionStatus.WithLabelValues("disconnected").Set(0)

	nc.SetReconnectHandler(func(conn *nats.Conn) {
		metrics.NatsConnectionStatus.WithLabelValues("connected").Set(1)
		metrics.NatsConnectionStatus.WithLabelValues("disconnected").Set(0)
	})

	nc.SetDisconnectErrHandler(func(conn *nats.Conn, err error) {
		metrics.NatsConnectionStatus.WithLabelValues("connected").Set(0)
		metrics.NatsConnectionStatus.WithLabelValues("disconnected").Set(1)
	})

	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}

	n.js = js
	return nil
}

func (n *Nats) Close(ctx context.Context) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	c := n.js.Conn()
	if c != nil {
		err := c.FlushWithContext(ctx)
		c.Close()
		return err
	}

	return nil
}

func (n *Nats) Accept(ctx context.Context, event cloudevents.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ack, err := n.js.PublishMsg(ctx, &nats.Msg{
		Data:    data,
		Subject: n.subject,
	})
	if err != nil {
		return err
	}

	slog.Debug("Published msg", "sequence number", ack.Sequence, "stream", ack.Stream)
	return nil
}
