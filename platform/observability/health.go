package observability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Probe reports whether one dependency is usable right now. It must be cheap
// and must respect the context deadline: a probe that blocks turns a readiness
// check into an outage.
type Probe func(context.Context) error

// Health tracks liveness and readiness.
//
// The two are deliberately different questions. Liveness asks whether the
// process is running at all, and only a restart can fix a negative answer.
// Readiness asks whether it should receive traffic right now, and a database
// that is briefly unreachable should take the instance out of rotation
// without killing it.
type Health struct {
	mu       sync.RWMutex
	probes   map[string]Probe
	shutdown bool
}

// NewHealth creates an empty health tracker.
func NewHealth() *Health {
	return &Health{probes: make(map[string]Probe)}
}

// Register adds a readiness probe under a name that appears in the /readyz
// response and in the shutdown logs.
func (h *Health) Register(name string, probe Probe) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.probes[name] = probe
}

// BeginShutdown marks the service as draining. Readiness fails from this
// moment on, which is what lets a load balancer stop sending new requests
// before the listener actually closes.
func (h *Health) BeginShutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.shutdown = true
}

// IsShuttingDown reports whether BeginShutdown has been called.
func (h *Health) IsShuttingDown() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.shutdown
}

// ErrShuttingDown is returned by Check once shutdown has begun.
var ErrShuttingDown = errors.New("service is shutting down")

// Check runs every probe and returns their individual results plus the
// combined error. Probes run sequentially: there are only a handful per
// service, and sequential execution keeps the failure report deterministic.
func (h *Health) Check(ctx context.Context) (map[string]string, error) {
	h.mu.RLock()
	shutdown := h.shutdown
	probes := make(map[string]Probe, len(h.probes))

	for name, probe := range h.probes {
		probes[name] = probe
	}

	h.mu.RUnlock()

	results := make(map[string]string, len(probes)+1)

	if shutdown {
		results["service"] = ErrShuttingDown.Error()

		return results, ErrShuttingDown
	}

	var failures []error

	for name, probe := range probes {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := probe(probeCtx)

		cancel()

		if err != nil {
			results[name] = err.Error()
			failures = append(failures, fmt.Errorf("%s: %w", name, err))

			continue
		}

		results[name] = "ok"
	}

	return results, errors.Join(failures...)
}
