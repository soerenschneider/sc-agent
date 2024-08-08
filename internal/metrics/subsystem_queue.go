package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemQueue = "queue"

var (
	QueueCapacity = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemQueue,
		Name:      "capacity_total",
		Help:      "Total capacity of the queue",
	})

	QueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemQueue,
		Name:      "entries_total",
		Help:      "Total entries of the queue",
	})

	QueueErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemQueue,
		Name:      "errors_total",
		Help:      "Total amount of errors while trying to enqueue notifications",
	}, []string{"operation"})
)
