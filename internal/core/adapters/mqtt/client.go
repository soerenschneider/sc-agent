package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

type Client struct {
	conf     *config.Mqtt
	client   mqtt.Client
	services *ports.Components
}

func NewMqttClient(conf *config.Mqtt, services *ports.Components) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(conf.Broker)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(60 * time.Second)
	opts.SetConnectRetry(true)
	opts.SetClientID(conf.ClientId)
	tlsConfig := conf.TlsConfig()
	if tlsConfig != nil {
		opts.SetTLSConfig(tlsConfig)
	}

	opts.OnConnectionLost = connectLostHandler
	opts.OnConnectAttempt = onConnectAttemptHandler
	opts.OnConnect = onConnectHandler
	opts.OnReconnecting = onReconnectHandler

	client := mqtt.NewClient(opts)

	return &Client{
		conf:     conf,
		client:   client,
		services: services,
	}, nil
}

func (c *Client) StartListener(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	token := c.client.Connect()
	finishedWithinTimeout := token.WaitTimeout(10 * time.Second)
	if token.Error() != nil || !finishedWithinTimeout {
		log.Error().Err(token.Error()).Str("component", "mqtt").Msg("Connection to broker failed, continuing in background")
	}

	router, err := NewTopicRouter(c.client, c.conf)
	if err != nil {
		return fmt.Errorf("could not build router: %w", err)
	}

	// Register handlers for different topic patterns
	powerstatusHandler := &PowerStatusHandler{services: c.services}
	router.MustRegisterHandler("power/shutdown", powerstatusHandler)
	router.MustRegisterHandler("power/reboot", powerstatusHandler)

	packageHandler := &PackagesHandler{services: c.services}
	router.MustRegisterHandler("packages/list", packageHandler)
	router.MustRegisterHandler("packages/upgrade", packageHandler)

	if err := router.Subscribe(); err != nil {
		return err
	}

	<-ctx.Done()
	log.Info().Str("component", "mqtt").Msg("Disconnecting from broker")
	c.client.Disconnect(3000)
	log.Info().Str("component", "mqtt").Msg("Stopping component")

	return nil
}

var mutex sync.Mutex

func connectLostHandler(client mqtt.Client, err error) {
	opts := client.OptionsReader()
	log.Warn().Err(err).Str("component", "mqtt").Any("brokers", opts.Servers()).Msg("Connection lost")
	metrics.MqttConnectionsLostTotal.Inc()
	mutex.Lock()
	defer mutex.Unlock()
	metrics.MqttBrokersConnectedTotal.Sub(1)
}

func onReconnectHandler(client mqtt.Client, opts *mqtt.ClientOptions) {
	mutex.Lock()
	metrics.MqttReconnectionsTotal.Inc()
	mutex.Unlock()
	log.Info().Str("component", "mqtt").Any("brokers", opts.Servers).Msg("Reconnecting")
}

func onConnectAttemptHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	log.Info().Str("component", "mqtt").Str("broker", broker.Host).Msg("Trying connecting to broker")
	return tlsCfg
}

func onConnectHandler(c mqtt.Client) {
	opts := c.OptionsReader()
	brokers := make([]string, 0, len(opts.Servers()))
	for _, broker := range opts.Servers() {
		brokers = append(brokers, broker.Host)
	}
	log.Info().Str("component", "mqtt").Strs("brokers", brokers).Msg("Successfully connected")
	mutex.Lock()
	metrics.MqttBrokersConnectedTotal.Add(1)
	mutex.Unlock()
}
