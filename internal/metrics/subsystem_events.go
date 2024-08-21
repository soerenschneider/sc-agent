package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const eventsSubsystem = "events"

var (
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
