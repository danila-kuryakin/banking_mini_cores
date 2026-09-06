package grpc

import (
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
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
	return grpc_server.NewServer(
		s.addr,
		s.logger,
		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				authv1.RegisterAuthServiceServer(r, s.authServer)
			},
		),
		grpc_server.WithUnaryInterceptors(
			interceptors.Validate(),
		),
		grpc_server.WithHandlerTimeout(s.timeout),
	)
}

func (s *GRPCServer) Addr() string {
	return s.addr
}
