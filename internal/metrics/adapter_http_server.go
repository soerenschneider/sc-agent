package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const httpServerSubsystem = "adapters_http"

var (
	StatusCode = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: httpServerSubsystem,
		Name:      "requests_status",
	}, []string{"status_code"})
)
