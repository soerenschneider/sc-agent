package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystemConditionalReboot = "conditional_reboot"

var (
	ProcessStartTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "start_timestamp_seconds",
	})

	Heartbeat = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "heartbeat_timestamp_seconds",
	})

	RebootIsPaused = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "reboot_paused_bool",
	})

	Version = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "version",
	}, []string{"version"})

	CheckerLastCheck = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "checker_last_check_timestamp_seconds",
	}, []string{"checker"})

	CheckerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "checker_status",
	}, []string{"checker", "status"})

	AgentState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "agent_state",
	}, []string{"state", "checker"})

	LastStateChange = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "agent_state_change_timestamp_seconds",
	}, []string{"state", "checker"})

	RebootErrors = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemConditionalReboot,
		Name:      "reboot_errors_total",
	})
)
