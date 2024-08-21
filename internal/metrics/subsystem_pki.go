package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemPki = "pki"

var (
	PkiCertPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemPki,
		Name:      "percent",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	PkiExpirationDate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemPki,
		Name:      "expiry_timestamp_seconds",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	PkiErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemPki,
		Name:      "errors_total",
		Help:      "Expiration date of the token",
	}, []string{"cn", "error"})

	PkiReadRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemPki,
		Name:      "requests_total",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	PkiRequestTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemPki,
		Name:      "requests_timestamp_seconds",
		Help:      "Expiration date of the token",
	}, []string{"cn"})
)
