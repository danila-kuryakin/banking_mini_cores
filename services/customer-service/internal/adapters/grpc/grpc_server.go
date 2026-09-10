package grpc

import (
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/service"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	addr           string
	customerServer *CustomerServer
	statusServer   *StatusServer
	logger         *slog.Logger
	timeout        time.Duration
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:           addr,
		customerServer: NewCustomerServer(service, logger),
		statusServer:   NewStatusServer(service, logger),
		logger:         logger,
		timeout:        timeout,
	}
}

func (s *GRPCServer) Run() error {
	return grpc_server.NewServer(
		s.addr,
		s.logger,
		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				customerv1.RegisterCustomerServiceServer(r, s.customerServer)
			},
			func(r grpc.ServiceRegistrar) {
				customerv1.RegisterCustomerStatusServiceServer(r, s.statusServer)
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
