// Package grpc_server is the shared inbound gRPC runtime: it opens the
// listener, registers the services the caller passes in, exposes reflection
// and the standard health protocol, and shuts everything down gracefully.
//
// It holds no business logic and no knowledge of any concrete service: the
// protobuf registration comes in as a RegisterFunc and the listen address as
// Config, which is what makes the package reusable from platform.
package grpc_server

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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

// NewServer creates the adapter. Use cases will be injected here as they are
// implemented.
func NewServer(addr string, logger *slog.Logger, register ...RegisterFunc) error {
	s := &Server{logger: logger}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s.server = grpc.NewServer()
	for _, reg := range register {
		reg(s.server)
	}

	// Рефлексия нужна, чтобы сервис можно было дёргать через grpcurl без .proto.
	reflection.Register(s.server)

	// Стандартный health-протокол grpc.health.v1.Health - его опрашивает monitor.
	// Пустое имя означает статус процесса целиком, именованное - конкретного сервиса.
	s.health = healthServer(s.server)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		s.logger.Info("shutting down")
		// Сначала объявляем себя NOT_SERVING, чтобы monitor увидел штатное
		// выключение, и только потом закрываем сервер.
		s.health.Shutdown()
		s.server.GracefulStop()
	}()

	s.logger.Info("grpc listening", "addr", lis.Addr().String())
	// Serve блокирует до GracefulStop - именно это держит процесс живым.
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func healthServer(grpcSrv *grpc.Server) *health.Server {
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
