package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const nftablesSubsystem = "nftables"

var (
	NftablesNoSuchSet = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: nftablesSubsystem,
		Name:      "set_not_found_errors_total",
	}, []string{"set"})

	NftablesErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: nftablesSubsystem,
		Name:      "add_element_errors_total",
	}, []string{"set"})
)
