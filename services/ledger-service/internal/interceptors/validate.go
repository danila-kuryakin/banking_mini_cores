// Package interceptors holds the gRPC middleware specific to ledger-service.
//
// Общие для всех сервисов интерцепторы (recovery, request id, logging,
// metrics, timeout) подключает platform/grpc_server автоматически - здесь
// только то, что знает про домен проводок.
package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validator - контракт, который реализуют запросы, умеющие себя проверять.
// Если позже подключить protovalidate, сгенерированные сообщения начнут
// удовлетворять ему сами, и этот интерцептор менять не придётся.
type validator interface {
	Validate() error
}

// Validate отсекает заведомо некорректный запрос до похода в usecase, чтобы
// хендлеры не начинались с одинаковых проверок полей, а клиент всегда получал
// codes.InvalidArgument, а не Internal откуда-то из глубины.
//
// Запросы, не реализующие validator, проходят насквозь: интерцептор ничего не
// требует от pb-сообщений.
func Validate() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if v, ok := req.(validator); ok {
			if err := v.Validate(); err != nil {
				// Текст ошибки валидации безопасно отдавать клиенту: он про
				// его же запрос. Внутренние ошибки так наружу не выносим.
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		return handler(ctx, req)
	}
}
