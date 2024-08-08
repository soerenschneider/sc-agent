package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ProcessStartTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "start_timestamp_seconds",
	})

	Heartbeat = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "heartbeat_timestamp_seconds",
	})

	RebootIsPaused = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "reboot_paused_bool",
	})

	Version = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "version",
	}, []string{"version"})

	CheckerLastCheck = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "checker",
		Name:      "last_check_timestamp_seconds",
	}, []string{"checker"})

	CheckerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "checker",
		Name:      "status",
	}, []string{"checker", "status"})

	AgentState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "agent",
		Name:      "state",
	}, []string{"state", "checker"})

	LastStateChange = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "agent",
		Name:      "state_change_timestamp_seconds",
	}, []string{"state", "checker"})

	RebootErrors = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "invocation_errors_total",
	})
)
