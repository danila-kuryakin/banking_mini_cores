package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	"github.com/danila-kuryakin/banking_mini_cores/platform/interceptors"
)

// requestID проставляет корреляционный id входящему HTTP-запросу: берёт
// x-request-id у клиента или генерирует новый. Дальше его подхватывает
// gatewayMetadata и кладёт в gRPC-метаданные, где его уже ждёт
// interceptors.RequestID на стороне сервиса - так один и тот же id виден в
// логах gateway и всех сервисов за ним.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(interceptors.RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
			// Пишем в заголовки самого запроса, а не только в контекст:
			// grpc-gateway отдаёт в WithMetadata именно *http.Request.
			r.Header.Set(interceptors.RequestIDHeader, id)
		}
		// Отдаём id клиенту - по нему он сможет сослаться на конкретный вызов.
		w.Header().Set(interceptors.RequestIDHeader, id)

		next.ServeHTTP(w, r)
	})
}

// gatewayMetadata переносит корреляционный id из HTTP-заголовка в метаданные
// исходящего gRPC-вызова. Штатный HeaderMatcher grpc-gateway пропускает только
// permanent-заголовки, x-request-id в их число не входит.
func gatewayMetadata(_ context.Context, r *http.Request) metadata.MD {
	id := r.Header.Get(interceptors.RequestIDHeader)
	if id == "" {
		return nil
	}

	return metadata.Pairs(interceptors.RequestIDHeader, id)
}

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"took", time.Since(start),
			"request_id", r.Header.Get(interceptors.RequestIDHeader),
		)
	})
}
