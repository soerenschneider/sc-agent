package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemUpdater = "release_watcher"

var (
	UpdateAvailable = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemUpdater,
		Name:      "update_available_bool",
		Help:      "Yields 1 if an update is available, otherwise 0",
	})

	UpdateCheckErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemUpdater,
		Name:      "update_check_errors_total",
		Help:      "Errors while checking for updates",
	})
)
