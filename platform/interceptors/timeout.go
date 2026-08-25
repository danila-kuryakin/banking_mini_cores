package interceptors

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Timeout гарантирует хендлеру верхнюю границу времени жизни. Без него
// зависший запрос к Postgres держит горутину и соединение из пула столько,
// сколько клиент готов ждать - то есть потенциально вечно.
//
// context.WithTimeout берёт минимум из родительского дедлайна и своего,
// так что более жёсткий дедлайн клиента интерцептор не ослабляет.
func Timeout(d time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()

		resp, err := handler(ctx, req)
		return resp, mapContextError(ctx, err)
	}
}

// mapContextError переводит голую ошибку контекста в gRPC-статус.
//
// Дедлайн, выставленный клиентом, транспорт grpc отображает сам, но наш
// собственный - нет: хендлер возвращает context.DeadlineExceeded, и клиент
// получает бессмысленный codes.Unknown. Поэтому дожимаем вручную.
func mapContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	// Если хендлер уже вернул осмысленный статус - он знает лучше, не трогаем.
	if status.Code(err) != codes.Unknown {
		return err
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "handler deadline exceeded")
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	default:
		return err
	}
}
