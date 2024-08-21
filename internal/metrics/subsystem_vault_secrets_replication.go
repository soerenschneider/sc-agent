package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemVaultSecretSyncer = "vault_secrets_replication"

var (
	SecretsCacheHit = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultSecretSyncer,
		Name:      "cache_hits_total",
		Help:      "Total number of cache hits",
	}, []string{"path"})

	SecretsRead = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultSecretSyncer,
		Name:      "secrets_read_total",
		Help:      "Total amount of secrets read",
	}, []string{"path"})

	SecretReplicationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemVaultSecretSyncer,
		Name:      "errors_total",
		Help:      "Total errors while replicating secrets",
	}, []string{"path", "component"})
)
