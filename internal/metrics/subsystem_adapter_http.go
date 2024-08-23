package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const adapterHttpSubsystem = "adapters_http"

var (
	AdapterHttpTlsErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   namespace,
		Subsystem:   adapterHttpSubsystem,
		Name:        "tls_errors_total",
		Help:        "Problems regarding TLS certificates",
		ConstLabels: nil,
	})
)
