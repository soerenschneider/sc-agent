package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const eventsSubsystem = "events"

var (
	EventQueueCapacity = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "queue_capacity_total",
	})

	EventQueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "queue_size_total",
	})

	EventReceivedTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "received_timestamp_seconds",
	}, []string{"source"})

	EventsReceived = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "received_total",
	}, []string{"source"})

	NatsConnectionStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "nats_connection_status",
	}, []string{"status"})

	NatsErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "nats_connection_errors_total",
	}, []string{"error"})

	MqttReconnectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "mqtt_reconnections_total",
	})

	MqttConnectionsLostTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "mqtt_connections_lost_total",
	})

	MqttBrokersConnectedTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "mqtt_connected_brokers_total",
	})

	RabbitMqErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "rabbitmq_errors_total",
	}, []string{"error"})

	RabbitMqDisconnects = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: eventsSubsystem,
		Name:      "rabbitmq_disconnects_total",
	}, []string{"type"})
)
