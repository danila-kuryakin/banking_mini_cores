package server

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/service"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

type GRPCServer struct {
	addr            string
	antifraudServer *AntifraudServer
	logger          *slog.Logger
	timeout         time.Duration
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:            addr,
		antifraudServer: NewAntifraudServer(service, logger),
		logger:          logger,
		timeout:         timeout,
	}
}

func (s *GRPCServer) Run() error {
	return grpc_server.NewServer(
		s.addr,
		s.logger,
		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				antifraudv1.RegisterAntifraudServiceServer(r, s.antifraudServer)
			},
		),
		grpc_server.WithUnaryInterceptors(
			interceptors.Validate(),
		),
		grpc_server.WithHandlerTimeout(s.timeout),
	)
}
