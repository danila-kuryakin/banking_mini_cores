// Package grpc_server is the shared inbound gRPC runtime: it opens the
// listener, registers the services the caller passes in, wires the platform
// interceptor chain, exposes reflection and the standard health protocol, and
// shuts everything down gracefully.
//
// It holds no business logic and no knowledge of any concrete service: the
// protobuf registration comes in as a RegisterFunc and the listen address as
// Config, which is what makes the package reusable from platform.
package grpc_server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/danila-kuryakin/banking_mini_cores/platform/interceptors"
)

// RegisterFunc регистрирует конкретную реализацию сервиса на уже созданном
// grpc.Server. Именно так вызывающая сторона (main) прокидывает сюда
// authv1.RegisterAuthServiceServer, а пакет остаётся независимым от pb.
type RegisterFunc func(grpc.ServiceRegistrar)

type Server struct {
	server *grpc.Server
	health *health.Server
	logger *slog.Logger
}

// NewServer поднимает gRPC-сервер с базовой цепочкой интерцепторов и блокирует
// до сигнала завершения.
func NewServer(addr string, logger *slog.Logger, opts ...Option) error {
	o := newOptions(opts...)

	if len(o.services) == 0 {
		// Иначе сервер молча поднимется пустым и будет отвечать Unimplemented
		// на всё - диагностировать это по логам неприятно.
		return fmt.Errorf("no services registered: pass grpc_server.WithServices(...)")
	}

	s := &Server{logger: logger}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s.server = grpc.NewServer(serverOptions(logger, o)...)

	for _, register := range o.services {
		register(s.server)
	}

	// Рефлексия нужна, чтобы сервис можно было дёргать через grpcurl без .proto.
	if !o.disableReflection {
		reflection.Register(s.server)
	}

	// Стандартный health-протокол grpc.health.v1.Health - его опрашивает monitor.
	// Пустое имя означает статус процесса целиком, именованное - конкретного сервиса.
	s.health = HealthServer(s.server)

	// NotifyContext сам снимает обработчик сигнала по stop() - в отличие от
	// голого signal.Notify в горутине, которая висела бы вечно, если Serve
	// вернётся с ошибкой, так и не дождавшись сигнала.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		s.shutdown(o.shutdownTimeout)
	}()

	s.logger.Info("grpc listening",
		"addr", lis.Addr().String(),
		"services", serviceNames(s.server),
	)
	// Serve блокирует до GracefulStop - именно это держит процесс живым.
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

// serverOptions собирает grpc.ServerOption: трассировку и обе цепочки
// интерцепторов - сначала базовые от платформы, затем интерцепторы сервиса.
func serverOptions(logger *slog.Logger, o options) []grpc.ServerOption {
	grpcOpts := o.grpcOpts

	// Трассировка в otelgrpc - это stats.Handler, а не интерцептор: он видит
	// события соединения и сообщений, до которых интерцептор не достаёт.
	if !o.disableTracing {
		grpcOpts = append(grpcOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	}

	unary := append(defaultUnaryInterceptors(logger, o), o.unary...)
	if len(unary) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(unary...))
	}

	stream := append(defaultStreamInterceptors(logger, o), o.stream...)
	if len(stream) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainStreamInterceptor(stream...))
	}

	return grpcOpts
}

// defaultUnaryInterceptors - базовая цепочка, которую получает каждый сервис.
// Порядок здесь и есть весь смысл функции:
//
//	Recovery  - снаружи всех, ловит панику в том числе из соседних интерцепторов;
//	RequestID - до логов и метрик, иначе им нечего писать в поле request_id;
//	Metrics   - снаружи Logging, чтобы латентность включала всё, что после;
//	Logging   - на обратном пути отрабатывает последним из базовых, поэтому
//	            видит код ответа уже после маппинга ошибок в сервисе;
//	Timeout   - вплотную к интерцепторам сервиса: дедлайн нужен работе,
//	            а не логированию её результата.
func defaultUnaryInterceptors(logger *slog.Logger, o options) []grpc.UnaryServerInterceptor {
	if o.disableInterceptors {
		return nil
	}

	chain := []grpc.UnaryServerInterceptor{
		interceptors.Recovery(logger),
		interceptors.RequestID(),
	}
	if !o.disableMetrics {
		chain = append(chain, interceptors.Metrics(o.metricsReg))
	}
	chain = append(chain, interceptors.Logging(logger))
	if o.handlerTimeout > 0 {
		chain = append(chain, interceptors.Timeout(o.handlerTimeout))
	}
	return chain
}

// defaultStreamInterceptors намеренно короче unary-цепочки: Timeout стриму
// противопоказан (Health/Watch и любой long-lived стрим живут дольше любого
// разумного дедлайна), а метрики по завершению стрима меряют не латентность
// запроса, а время жизни подписки.
func defaultStreamInterceptors(logger *slog.Logger, o options) []grpc.StreamServerInterceptor {
	if o.disableInterceptors {
		return nil
	}
	return []grpc.StreamServerInterceptor{
		interceptors.RecoveryStream(logger),
		interceptors.RequestIDStream(),
		interceptors.LoggingStream(logger),
	}
}

func (s *Server) shutdown(timeout time.Duration) {
	s.logger.Info("shutting down", "timeout", timeout)
	// Сначала объявляем себя NOT_SERVING, чтобы monitor увидел штатное
	// выключение, и только потом закрываем сервер.
	s.health.Shutdown()

	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("stopped gracefully")
	case <-time.After(timeout):
		// GracefulStop ждёт завершения всех активных RPC. Если клиент держит
		// долгий вызов, ждать можно бесконечно - поэтому рубим по таймауту.
		s.logger.Warn("graceful shutdown timed out, forcing stop")
		s.server.Stop()
	}
}

func HealthServer(grpcSrv *grpc.Server) *health.Server {
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	// Имена сервисов берём у самого grpc.Server - так health знает про всё, что
	// зарегистрировали через RegisterFunc, без импорта pb-пакетов.
	for name := range grpcSrv.GetServiceInfo() {
		healthSrv.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
	}
	return healthSrv
}

func serviceNames(grpcSrv *grpc.Server) []string {
	info := grpcSrv.GetServiceInfo()
	names := make([]string, 0, len(info))
	for name := range info {
		names = append(names, name)
	}
	return names
}
