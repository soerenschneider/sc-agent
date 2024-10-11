package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const packagesSubsystem = "packages"

var (
	UpdatesAvailableBool = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: packagesSubsystem,
		Name:      "updates_available_bool",
	})

	UpdatesAvailable = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: packagesSubsystem,
		Name:      "updates_available_total",
	})

	UpdateCheckAvailableTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: packagesSubsystem,
		Name:      "updates_available_check_timestamp_seconds",
	})

	PackagesInstalled = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: packagesSubsystem,
		Name:      "installed_total",
	})

	DnfErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: packagesSubsystem,
		Name:      "errors_total",
	}, []string{"command"})
)
