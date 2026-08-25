// Package interceptors holds the gRPC middleware that every service in the
// monorepo gets for free: panic recovery, request correlation, logging,
// metrics and a handler deadline.
//
// Nothing here knows about a concrete domain - that is the whole point.
// Service-specific middleware (authentication, validation, domain error
// mapping) lives inside the service and is appended to this chain through
// grpc_server.WithUnaryInterceptors.
package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
)

// splitFullMethod разбирает "/auth.v1.AuthService/Login" на имя сервиса и имя
// метода. Нужно и метрикам (иначе кардинальность лейблов уезжает), и логам.
// На нестандартном формате возвращает его целиком как метод - лучше кривой
// лейбл, чем паника в middleware.
func splitFullMethod(fullMethod string) (service, method string) {
	trimmed := strings.TrimPrefix(fullMethod, "/")
	i := strings.LastIndex(trimmed, "/")
	if i < 0 {
		return "unknown", trimmed
	}
	return trimmed[:i], trimmed[i+1:]
}

// wrappedStream подменяет контекст у стрима. grpc.ServerStream отдаёт контекст
// только на чтение, поэтому единственный способ прокинуть в стрим-хендлер
// обогащённый ctx - обернуть сам стрим.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

func wrapStream(ss grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &wrappedStream{ServerStream: ss, ctx: ctx}
}
