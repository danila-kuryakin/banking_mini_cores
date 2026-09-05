package grpc

import (
	"errors"
	"fmt"
	"net"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type HealthServer struct {
	addr   string
	server *grpc.Server
	health *health.Server
}

func NewHealthServer(addr string) *HealthServer {
	grpcSrv := grpc.NewServer()
	reflection.Register(grpcSrv)
	healthSrv := grpc_server.HealthServer(grpcSrv)

	return &HealthServer{
		addr:   addr,
		server: grpcSrv,
		health: healthSrv,
	}
}

func (s *HealthServer) Run() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}

	if err := s.server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (s *HealthServer) SetServingStatus(service string, serving bool) {
	status := healthpb.HealthCheckResponse_NOT_SERVING
	if serving {
		status = healthpb.HealthCheckResponse_SERVING
	}

	s.health.SetServingStatus(service, status)
}

func (s *HealthServer) Shutdown() {
	s.health.Shutdown()
	s.server.GracefulStop()
}

func (s *HealthServer) Addr() string {
	return s.addr
}
