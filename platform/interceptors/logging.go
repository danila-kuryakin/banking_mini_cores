package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type loggerKey struct{}

// Logging пишет ровно одну строку на вызов и кладёт в контекст логгер,
// уже помеченный request_id и методом.
//
// Тело запроса не логируется намеренно: в запросах вроде auth.v1.LoginRequest
// пароль лежит открытым текстом, и попав в лог он оттуда уже не исчезнет.
// Если нужны поля конкретного запроса - логируйте их точечно в хендлере.
func Logging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ctx = withLogger(ctx, log, info.FullMethod)

		resp, err := handler(ctx, req)

		logCall(ctx, LoggerFrom(ctx, log), start, err)
		return resp, err
	}
}

func LoggingStream(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := withLogger(ss.Context(), log, info.FullMethod)

		err := handler(srv, wrapStream(ss, ctx))

		logCall(ctx, LoggerFrom(ctx, log), start, err)
		return err
	}
}

// LoggerFrom возвращает логгер вызова с уже проставленными request_id и
// методом. fallback используется, если интерцептор не отработал.
func LoggerFrom(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return fallback
}

func withLogger(ctx context.Context, log *slog.Logger, fullMethod string) context.Context {
	service, method := splitFullMethod(fullMethod)
	l := log.With(
		"grpc_service", service,
		"grpc_method", method,
		"request_id", RequestIDFrom(ctx),
	)
	return context.WithValue(ctx, loggerKey{}, l)
}

func logCall(ctx context.Context, log *slog.Logger, start time.Time, err error) {
	code := status.Code(err)
	attrs := []any{
		"code", code.String(),
		"duration_ms", time.Since(start).Milliseconds(),
	}
	if err != nil {
		attrs = append(attrs, "err", err.Error())
	}

	log.Log(ctx, levelFor(code), "rpc handled", attrs...)
}

// levelFor разделяет "клиент сделал что-то не то" и "у нас сломалось".
// Сравнивать коды по величине нельзя: Unauthenticated (16) численно больше
// Internal (13), но это не наша авария - поэтому явный перечень.
func levelFor(code codes.Code) slog.Level {
	switch code {
	case codes.OK:
		return slog.LevelInfo
	case codes.Unknown,
		codes.DeadlineExceeded,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss:
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
