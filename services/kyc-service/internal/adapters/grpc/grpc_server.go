package grpc

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

type GRPCServer struct {
	addr      string
	kycServer *KycServer
	logger    *slog.Logger
	timeout   time.Duration
}

func NewGRPCServer(addr string, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:      addr,
		kycServer: NewKycServer(logger),
		logger:    logger,
		timeout:   timeout,
	}
}

func (s *GRPCServer) Run() error {
	return grpc_server.NewServer(
		s.addr,
		s.logger,
		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				kycv1.RegisterKycServiceServer(r, s.kycServer)
			},
		),
		grpc_server.WithHandlerTimeout(s.timeout),
	)
}

func (s *GRPCServer) Addr() string {
	return s.addr
}
