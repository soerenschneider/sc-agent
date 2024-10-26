package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemRebootManager = "reboot_manager"

var (
	ProcessStartTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "start_timestamp_seconds",
	})

	Heartbeat = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "heartbeat_timestamp_seconds",
	})

	RebootIsPaused = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "reboot_paused_bool",
	})

	Version = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "version",
	}, []string{"version"})

	CheckerLastCheck = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "checker_last_check_timestamp_seconds",
	}, []string{"checker"})

	CheckerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "checker_status",
	}, []string{"checker", "status"})

	AgentState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "agent_state",
	}, []string{"state", "checker"})

	LastStateChange = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "agent_state_change_timestamp_seconds",
	}, []string{"state", "checker"})

	RebootErrors = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemRebootManager,
		Name:      "reboot_errors_total",
	})
)
