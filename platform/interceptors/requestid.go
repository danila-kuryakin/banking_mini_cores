package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// RequestIDHeader - метаданные, в которых приходит корреляционный id. Тот же
// ключ проставляет api-gateway, поэтому запрос прослеживается сквозь всю
// цепочку сервисов.
const RequestIDHeader = "x-request-id"

// Тип-ключ вместо строки: так значение из контекста не может перетереть
// чужой пакет, положивший туда строку "request_id".
type requestIDKey struct{}

// RequestID достаёт корреляционный id из метаданных или генерирует новый и
// кладёт в контекст. Ставится сразу после Recovery - всё, что логируется
// дальше, уже должно иметь id.
func RequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(withRequestID(ctx), req)
	}
}

func RequestIDStream() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, wrapStream(ss, withRequestID(ss.Context())))
	}
}

// RequestIDFrom возвращает корреляционный id вызова. Пустая строка означает,
// что интерцептор не отработал (например, вызов идёт не через gRPC).
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func withRequestID(ctx context.Context) context.Context {
	id := incomingRequestID(ctx)
	if id == "" {
		id = uuid.NewString()
	}
	return context.WithValue(ctx, requestIDKey{}, id)
}

func incomingRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	// md.Get сам приводит ключ к нижнему регистру, так что регистр заголовка
	// на стороне клиента значения не имеет.
	if v := md.Get(RequestIDHeader); len(v) > 0 {
		return v[0]
	}
	return ""
}
