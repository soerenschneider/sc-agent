package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemVaultApprole = "vault_approle"

var (
	SecretIdTtl = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_ttl_seconds",
		Help:      "Expiration date of the token",
	}, []string{"secret_id_file"})

	SecretIdPercentage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_ttl_percent",
		Help:      "Expiration date of the token",
	}, []string{"secret_id_file"})

	SecretIdRotationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_rotation_errors_total",
		Help:      "Expiration date of the token",
	}, []string{"secret_id_file", "error"})
)
