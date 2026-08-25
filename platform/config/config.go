// Package config loads service configuration from the environment.
//
// Twelve-factor, no config files: a container gets its settings from env vars
// and nothing else. Every lookup goes through this package so that a missing
// required variable fails loudly at startup rather than as a nil dereference
// on the first request.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Base is the configuration every service has, whatever it does.
type Base struct {
	// ServiceName labels logs, metrics and traces. Defaults to the value the
	// caller passes to LoadBase.
	ServiceName string

	// Env is dev, staging or prod. Only dev turns on human-readable logs and
	// gRPC server reflection.
	Env string

	// LogLevel is debug, info, warn or error.
	LogLevel string

	// GRPCAddr is the listen address of the gRPC server, e.g. ":50051".
	GRPCAddr string

	// AdminAddr is the listen address of the admin HTTP server that serves
	// /metrics, /healthz and /readyz. Kept off the gRPC port so it can be
	// exposed to Prometheus without exposing the API.
	AdminAddr string

	// OTLPEndpoint is the OpenTelemetry collector, host:port, gRPC. Empty
	// disables tracing, which is what unit tests want.
	OTLPEndpoint string

	// TraceSampleRatio is the head sampling ratio, 0.0 to 1.0.
	TraceSampleRatio float64

	// ShutdownTimeout bounds graceful shutdown before connections are cut.
	ShutdownTimeout time.Duration
}

// IsDev reports whether the service is running in the local development
// environment.
func (b Base) IsDev() bool { return b.Env == "dev" }

// LoadBase reads the common configuration, using serviceName as the default
// service name and defaultGRPCPort as the default gRPC port.
func LoadBase(serviceName string, defaultGRPCPort int) (Base, error) {
	sampleRatio, err := Float("OTEL_TRACE_SAMPLE_RATIO", 1.0)
	if err != nil {
		return Base{}, err
	}

	shutdown, err := Duration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Base{}, err
	}

	return Base{
		ServiceName:      String("SERVICE_NAME", serviceName),
		Env:              String("ENV", "dev"),
		LogLevel:         String("LOG_LEVEL", "info"),
		GRPCAddr:         String("GRPC_ADDR", fmt.Sprintf(":%d", defaultGRPCPort)),
		AdminAddr:        String("ADMIN_ADDR", ":9090"),
		OTLPEndpoint:     String("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		TraceSampleRatio: sampleRatio,
		ShutdownTimeout:  shutdown,
	}, nil
}

// String returns the value of key, or def when it is unset or empty.
func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}

	return def
}

// MustString returns the value of key, or an error when it is unset. Use it
// for anything the service cannot invent a safe default for, such as a
// database DSN or a signing key.
func MustString(key string) (string, error) {
	v := String(key, "")
	if v == "" {
		return "", fmt.Errorf("config: required environment variable %s is not set", key)
	}

	return v, nil
}

// Int returns the value of key parsed as an integer, or def when unset.
func Int(key string, def int) (int, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer, got %q: %w", key, raw, err)
	}

	return v, nil
}

// Float returns the value of key parsed as a float, or def when unset.
func Float(key string, def float64) (float64, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a number, got %q: %w", key, raw, err)
	}

	return v, nil
}

// Bool returns the value of key parsed as a boolean, or def when unset.
func Bool(key string, def bool) (bool, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("config: %s must be a boolean, got %q: %w", key, raw, err)
	}

	return v, nil
}

// Duration returns the value of key parsed as a Go duration such as "15s", or
// def when unset.
func Duration(key string, def time.Duration) (time.Duration, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a duration, got %q: %w", key, raw, err)
	}

	return v, nil
}

// StringSlice returns the value of key split on commas, or def when unset.
func StringSlice(key string, def []string) []string {
	raw := String(key, "")
	if raw == "" {
		return def
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return def
	}

	return out
}
