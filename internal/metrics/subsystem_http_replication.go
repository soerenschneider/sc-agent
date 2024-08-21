package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemHttpReplication = "http_replication"

var (
	HttpReplicationFileHash = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemHttpReplication,
		Name:      "hash_value_total",
		Help:      "Expiration date of the token",
	}, []string{"id"})
)
