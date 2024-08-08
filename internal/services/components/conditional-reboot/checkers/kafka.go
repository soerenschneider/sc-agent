package checkers

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"go.uber.org/multierr"
)

type KafkaChecker struct {
	brokers   []string
	topic     string
	partition int
	groupId   string

	acceptedKeys []string

	reader *kafka.Reader

	certFile string
	keyFile  string
}

type KafkaOpts func(checker *KafkaChecker) error

func NewKafkaChecker(brokers []string, topic string, opts ...KafkaOpts) (*KafkaChecker, error) {
	c := &KafkaChecker{
		brokers:      brokers,
		topic:        topic,
		acceptedKeys: getDefaultAcceptedKeys(),
	}

	var errs error
	for _, opt := range opts {
		if err := opt(c); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return c, errs
}

func getDefaultAcceptedKeys() (acceptedKeys []string) {
	systemHostname, err := os.Hostname()
	if err != nil {
		log.Warn().Err(err).Msg("could not auto-detect system hostname for kafka's accepted keys")
		return
	}

	acceptedKeys = append(acceptedKeys, systemHostname)
	return
}

func (c *KafkaChecker) Start() error {
	c.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:   c.brokers,
		Topic:     c.topic,
		Partition: c.partition,
		MaxBytes:  10e6,
		GroupID:   c.groupId,
	})

	return nil
}

func (c *KafkaChecker) IsHealthy(ctx context.Context) (bool, error) {
	for {
		m, err := c.reader.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))
	}

	if err := c.reader.Close(); err != nil {
		return false, err
	}

	return false, nil
}
