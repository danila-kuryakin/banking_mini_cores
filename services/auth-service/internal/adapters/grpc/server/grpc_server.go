package server

import (
	"log/slog"
	"time"

	platform_grpc "github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	addr       string
	authServer *AuthServer
	logger     *slog.Logger
	timeout    time.Duration
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:       addr,
		authServer: NewAuthServer(service, logger, timeout),
		logger:     logger,
		timeout:    timeout,
	}
}

func (s *GRPCServer) Run() error {
	return platform_grpc.NewServer(
		s.addr,
		s.logger,
		platform_grpc.WithServices(
			func(r grpc.ServiceRegistrar) {
				authv1.RegisterAuthServiceServer(r, s.authServer)
			},
		),
		platform_grpc.WithUnaryInterceptors(
			interceptors.Validate(),
		),
		platform_grpc.WithHandlerTimeout(s.timeout),
	)
}

func (s *GRPCServer) Addr() string {
	return s.addr
}
