package observability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// AdminServer serves the operational endpoints on a port separate from the
// API: /metrics for Prometheus, /healthz for liveness, /readyz for readiness.
//
// Separate port, not a separate path on the API port, so that scraping and
// probing never traverse the authentication and rate-limiting that guards the
// business API, and so the admin surface can stay off any public ingress.
type AdminServer struct {
	server *http.Server
	logger *slog.Logger
}

// NewAdminServer builds the admin server. addr is a listen address such as
// ":9081".
func NewAdminServer(addr string, metrics *Metrics, health *Health, logger *slog.Logger) *AdminServer {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{
		// A broken collector should be visible as a 500 on the scrape rather
		// than as a silently truncated metric set.
		ErrorHandling: promhttp.HTTPErrorOnError,
	}))

	// Liveness: the process is up and its HTTP stack answers. It deliberately
	// does not consult dependencies, or a database blip would get the
	// container killed instead of merely drained.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		results, err := health.Check(r.Context())

		status := http.StatusOK
		body := map[string]any{"status": "ready", "checks": results}

		if err != nil {
			status = http.StatusServiceUnavailable
			body["status"] = "not ready"
			body["error"] = err.Error()
		}

		writeJSON(w, status, body)
	})

	return &AdminServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		logger: logger,
	}
}

// Addr returns the configured listen address.
func (s *AdminServer) Addr() string { return s.server.Addr }

// Run serves until the server is shut down. A clean shutdown returns nil.
func (s *AdminServer) Run() error {
	s.logger.Info("admin server listening", slog.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("admin server: %w", err)
	}

	return nil
}

// Shutdown stops the admin server, waiting for in-flight scrapes.
func (s *AdminServer) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown admin server: %w", err)
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The status line is already written; there is nothing left to do but
		// let the scrape or probe fail on a truncated body.
		return
	}
}
