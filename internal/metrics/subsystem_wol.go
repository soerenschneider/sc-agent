package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const wolSubsystem = "wol"

var (
	WolNoSuchAlias = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: wolSubsystem,
		Name:      "alias_not_found_errors_toal",
	}, []string{"alias"})

	WolFailed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: wolSubsystem,
		Name:      "sending_package_failed_total",
	}, []string{"alias"})
)
