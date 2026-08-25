package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Recovery превращает панику в хендлере в codes.Internal вместо падения всего
// процесса. Всегда стоит первым в цепочке: так он накрывает не только хендлер,
// но и все остальные интерцепторы.
func Recovery(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// Именованный err обязателен: без него присваивание из defer не попадёт
		// в возвращаемое значение.
		defer func() {
			if r := recover(); r != nil {
				logPanic(ctx, log, info.FullMethod, r)
				resp, err = nil, status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// RecoveryStream - то же самое для стримовых методов.
func RecoveryStream(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logPanic(ss.Context(), log, info.FullMethod, r)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(srv, ss)
	}
}

func logPanic(ctx context.Context, log *slog.Logger, fullMethod string, r any) {
	// Стек снимаем прямо здесь, внутри defer: после выхода из recover он уже
	// не тот, по которому можно найти причину.
	log.ErrorContext(ctx, "panic recovered",
		"method", fullMethod,
		"panic", r,
		"stack", string(debug.Stack()),
		"request_id", RequestIDFrom(ctx),
	)
}
