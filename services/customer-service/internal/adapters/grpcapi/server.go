// Package grpc_server is the inbound gRPC adapter of auth-service: it translates
// protobuf messages into use-case calls and domain errors back into gRPC
// statuses.
//
// It holds no business logic. Everything it will do is delegate to
// internal/app, which is why the struct below is currently empty.
package grpcapi

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	service "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters"
	gwconfig "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/config"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	customerv1.UnimplementedCustomerServiceServer
}

// NewServer creates the adapter. Use cases will be injected here as they are
// implemented.
func NewServer(cfg gwconfig.Config, log *slog.Logger) error {
	lis, err := net.Listen("tcp", cfg.ServerAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ServerAddr, err)
	}

	srv := grpc.NewServer()
	customerv1.RegisterCustomerServiceServer(srv, service.NewCustomer(log))
	// Рефлексия нужна, чтобы сервис можно было дёргать через grpcurl без .proto.
	reflection.Register(srv)

	// Стандартный health-протокол grpc.health.v1.Health - его опрашивает monitor.
	// Пустое имя означает статус процесса целиком, именованное - конкретного сервиса.
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(customerv1.CustomerService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Info("shutting down")
		// Сначала объявляем себя NOT_SERVING, чтобы monitor увидел штатное
		// выключение, и только потом закрываем сервер.
		healthSrv.Shutdown()
		srv.GracefulStop()
	}()

	log.Info("grpc listening", "addr", lis.Addr().String())
	// Serve блокирует до GracefulStop - именно это держит процесс живым.
	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
