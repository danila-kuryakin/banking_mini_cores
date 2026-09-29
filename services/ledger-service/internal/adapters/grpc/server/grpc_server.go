package server

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/app/service"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

type GRPCServer struct {
	addr         string
	ledgerServer *LedgerServer
	logger       *slog.Logger
	timeout      time.Duration
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:         addr,
		ledgerServer: NewLedgerServer(service, logger),
		logger:       logger,
		timeout:      timeout,
	}
}

func (s *GRPCServer) Run() error {
	return grpc_server.NewServer(
		s.addr,
		s.logger,
		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				ledgerv1.RegisterLedgerServiceServer(r, s.ledgerServer)
			},
		),
		grpc_server.WithUnaryInterceptors(
			interceptors.Validate(),
		),
		grpc_server.WithHandlerTimeout(s.timeout),
	)
}
