package grpc

import (
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	addr           string
	customerServer *CustomerServer
	logger         *slog.Logger
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger) *GRPCServer {
	return &GRPCServer{
		addr:           addr,
		customerServer: NewCustomerServer(service, logger, time.Second),
		logger:         logger,
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
		),
		grpc_server.WithUnaryInterceptors(
			interceptors.Validate(),
		),
		grpc_server.WithHandlerTimeout(domain.DEFAULT_REQEST_TIMEOUT),
	)
}

func (s *GRPCServer) Addr() string {
	return s.addr
}
