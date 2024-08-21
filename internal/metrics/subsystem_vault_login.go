package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemVaultRenewal = "vault_renewal"

var (
	TokenTtl = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultRenewal,
		Name:      "token_ttl_seconds",
		Help:      "Expiration date of the token",
	}, []string{"name"})

	TokenPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultRenewal,
		Name:      "token_percent",
		Help:      "Expiration date of the token",
	}, []string{"name"})

	VaultLoginErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultRenewal,
		Name:      "login_errors_total",
		Help:      "Expiration date of the token",
	}, []string{"name"})

	VaultLogins = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultRenewal,
		Name:      "logins_total",
		Help:      "Expiration date of the token",
	}, []string{"name"})

	VaultTokenRenewErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultRenewal,
		Name:      "renew_errors_total",
		Help:      "Expiration date of the token",
	}, []string{"name"})

	VaultTokenRenewals = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultRenewal,
		Name:      "renewals_total",
		Help:      "Expiration date of the token",
	}, []string{"name"})
)
