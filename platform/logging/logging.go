// Package logging builds the structured logger every service uses and carries
// it through the request context.
//
// Output is JSON so that logs are queryable, and every line emitted from
// within a request automatically carries the request id, trace id and
// authenticated subject: correlation must not depend on a developer
// remembering to add a field.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/danila-kuryakin/banking-kyc/platform/ctxmeta"
)

type ctxKey struct{}

// New builds the root logger for a service.
//
// level accepts debug, info, warn or error, case-insensitively; anything else
// falls back to info rather than failing startup over a typo in an env var.
// When pretty is true the output is human-readable text for local runs.
func New(serviceName, level string, pretty bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if pretty {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(slog.String("service", serviceName))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Into returns a context carrying logger.
func Into(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// From returns the logger stored in ctx, or slog.Default when there is none,
// so a caller never has to nil-check before logging.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}

	return slog.Default()
}

// WithRequestAttrs derives a logger annotated with everything known about the
// current request: correlation ids from ctxmeta and, when the context is
// inside a sampled span, the OpenTelemetry trace and span ids. Attaching the
// trace id here is what lets a log line jump to its trace in Jaeger.
func WithRequestAttrs(ctx context.Context, logger *slog.Logger) *slog.Logger {
	attrs := make([]any, 0, 8)

	if v := ctxmeta.RequestID(ctx); v != "" {
		attrs = append(attrs, slog.String("request_id", v))
	}

	if v := ctxmeta.UserID(ctx); v != "" {
		attrs = append(attrs, slog.String("user_id", v))
	}

	if v := ctxmeta.CustomerID(ctx); v != "" {
		attrs = append(attrs, slog.String("customer_id", v))
	}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	if len(attrs) == 0 {
		return logger
	}

	return logger.With(attrs...)
}
