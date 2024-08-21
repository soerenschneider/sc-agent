package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemAcme = "acme"

var (
	AcmeCertPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemAcme,
		Name:      "percent",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	AcmeExpirationDate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemAcme,
		Name:      "expiry_timestamp_seconds",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	AcmeErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemAcme,
		Name:      "errors_total",
		Help:      "Expiration date of the token",
	}, []string{"cn", "error"})

	AcmeReadRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemAcme,
		Name:      "requests_total",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	AcmeRequestsTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemAcme,
		Name:      "requests_timestamp_seconds",
		Help:      "Expiration date of the token",
	}, []string{"cn"})
)
