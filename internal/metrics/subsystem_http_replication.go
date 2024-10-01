package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemHttpReplication = "http_replication"

var (
	HttpReplicationTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemHttpReplication,
		Name:      "timestamp_seconds",
		Help:      "Timestamp of the last attempt to replicate an item",
	}, []string{"id"})

	HttpReplicationRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemHttpReplication,
		Name:      "requests_total",
		Help:      "Total number of requests to replicate an item",
	}, []string{"id"})

	HttpReplicationFileHash = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemHttpReplication,
		Name:      "hash_value_total",
		Help:      "Hash value of the content",
	}, []string{"id"})

	HttpReplicationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemHttpReplication,
		Name:      "errors_total",
		Help:      "Errors while replicating",
	}, []string{"id", "error"})
)
