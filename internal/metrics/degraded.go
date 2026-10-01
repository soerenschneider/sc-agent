package metrics

import (
	"slices"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	Degraded = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "degraded",
		Help:      "Whether sc-agent is running in degraded mode, i.e. at least one component is not operational (1) or not (0)",
	})

	ComponentDegraded = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "component_degraded",
		Help:      "Whether a component is not operational (1) or operational (0)",
	}, []string{"component"})

	degradedComponents = map[string]struct{}{}
	degradedMutex      sync.Mutex
)

// SetComponentDegraded marks a component as not operational.
func SetComponentDegraded(component string) {
	degradedMutex.Lock()
	defer degradedMutex.Unlock()

	degradedComponents[component] = struct{}{}
	ComponentDegraded.WithLabelValues(component).Set(1)
	Degraded.Set(1)
}

// SetComponentHealthy marks a component as operational.
func SetComponentHealthy(component string) {
	degradedMutex.Lock()
	defer degradedMutex.Unlock()

	delete(degradedComponents, component)
	ComponentDegraded.WithLabelValues(component).Set(0)
	if len(degradedComponents) == 0 {
		Degraded.Set(0)
	}
}

// DegradedComponents returns the sorted names of all components that are currently not operational.
func DegradedComponents() []string {
	degradedMutex.Lock()
	defer degradedMutex.Unlock()

	ret := make([]string, 0, len(degradedComponents))
	for component := range degradedComponents {
		ret = append(ret, component)
	}
	slices.Sort(ret)
	return ret
}
