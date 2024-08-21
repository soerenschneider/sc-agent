package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemSsh = "ssh"

var (
	SshRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemSsh,
		Name:      "requests_total",
		Help:      "Expiration date of the token",
	}, []string{"public_key_file"})

	SshRequestTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemSsh,
		Name:      "requests_timestamp_seconds",
		Help:      "Expiration date of the token",
	}, []string{"public_key_file"})

	SshExpirationDate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemSsh,
		Name:      "expiry_timestamp_seconds",
		Help:      "Expiration date of the token",
	}, []string{"cn"})

	SshCertPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemSsh,
		Name:      "percent",
		Help:      "Expiration date of the token",
	}, []string{"public_key_file"})

	SshErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemSsh,
		Name:      "errors_total",
		Help:      "Expiration date of the token",
	}, []string{"public_key_file", "error"})
)
