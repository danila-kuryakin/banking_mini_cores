package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics owns a service-private Prometheus registry.
//
// A private registry rather than the default global one: it keeps test runs
// from colliding on duplicate registration, and it makes the exported set an
// explicit decision instead of whatever happened to be imported.
type Metrics struct {
	registry *prometheus.Registry
}

// NewMetrics creates a registry preloaded with the Go runtime and process
// collectors, labelled with the service name.
func NewMetrics(serviceName string) *Metrics {
	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		buildInfo(serviceName),
	)

	return &Metrics{registry: registry}
}

// Registry exposes the registry so that packages owning their own collectors,
// such as interceptors, can register them.
func (m *Metrics) Registry() *prometheus.Registry { return m.registry }

// MustRegister registers collectors and panics on conflict. Called during
// startup, where a duplicate metric is a programming error worth failing on.
func (m *Metrics) MustRegister(cs ...prometheus.Collector) { m.registry.MustRegister(cs...) }

func buildInfo(serviceName string) prometheus.Collector {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_build_info",
		Help: "Always 1. Its labels identify the running service.",
	}, []string{"service"})

	g.WithLabelValues(serviceName).Set(1)

	return g
}
