package interceptors

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Metrics считает вызовы и латентность по (сервис, метод, код ответа).
// Коллекторы регистрируются в reg; передайте prometheus.DefaultRegisterer,
// если экспортируете метрики стандартным promhttp.Handler().
func Metrics(reg prometheus.Registerer) grpc.UnaryServerInterceptor {
	m := newMetrics(reg)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		service, method := splitFullMethod(info.FullMethod)
		start := time.Now()

		resp, err := handler(ctx, req)

		code := status.Code(err).String()
		m.handled.WithLabelValues(service, method, code).Inc()
		m.duration.WithLabelValues(service, method, code).Observe(time.Since(start).Seconds())

		return resp, err
	}
}

type metrics struct {
	handled  *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	labels := []string{"grpc_service", "grpc_method", "grpc_code"}

	return &metrics{
		handled: mustRegister(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_server_handled_total",
			Help: "Total number of RPCs completed on the server.",
		}, labels)),
		duration: mustRegister(reg, prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "grpc_server_handling_seconds",
			Help: "Histogram of response latency of RPCs handled by the server.",
			// Границы подобраны под сервис, ходящий в БД: интересен участок
			// от единиц миллисекунд до нескольких секунд.
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, labels)),
	}
}

// mustRegister переиспользует уже зарегистрированный коллектор вместо паники.
// Иначе второй вызов NewServer в одном процессе (например, в тестах) валит
// приложение на AlreadyRegisteredError.
func mustRegister[T prometheus.Collector](reg prometheus.Registerer, c T) T {
	if err := reg.Register(c); err != nil {
		var already prometheus.AlreadyRegisteredError
		if errors.As(err, &already) {
			if existing, ok := already.ExistingCollector.(T); ok {
				return existing
			}
		}
		panic(err)
	}
	return c
}
