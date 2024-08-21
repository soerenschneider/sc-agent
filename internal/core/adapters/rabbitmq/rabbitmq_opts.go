package rabbitmq

import "errors"

func WithConsumerName(consumer string) RabbitMqOpts {
	return func(listener *RabbitMqEventListener) error {
		if len(consumer) == 0 {
			return errors.New("consumer may not be empty")
		}
		listener.consumerName = consumer
		return nil
	}
}

func WithTLS(certFile, keyFile string) RabbitMqOpts {
	return func(listener *RabbitMqEventListener) error {
		if len(certFile) == 0 {
			return errors.New("empty certfile")
		}

		if len(keyFile) == 0 {
			return errors.New("empty keyfile")
		}

		listener.connection.CertFile = certFile
		listener.connection.KeyFile = keyFile
		return nil
	}
}
