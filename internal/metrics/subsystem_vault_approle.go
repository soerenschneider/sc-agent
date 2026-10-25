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

	SecretIdExpiration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_expiration_timestamp_seconds",
		Help:      "Expiration of the secret_id as unix timestamp, absent for secret_ids without expiration",
	}, []string{"secret_id_file"})

	SecretIdRotationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_rotation_errors_total",
		Help:      "Expiration date of the token",
	}, []string{"secret_id_file", "error"})

	SecretIdBootstraps = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_bootstraps_total",
		Help:      "Wrapped secret_ids successfully fetched from approle-courier",
	}, []string{"secret_id_file"})

	SecretIdBootstrapErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultApprole,
		Name:      "secret_id_bootstrap_errors_total",
		Help:      "Errors while fetching a wrapped secret_id from approle-courier",
	}, []string{"secret_id_file", "error"})
)
