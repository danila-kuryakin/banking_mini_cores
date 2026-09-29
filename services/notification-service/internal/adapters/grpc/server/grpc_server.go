package server

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/service"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

type GRPCServer struct {
	addr               string
	notificationServer *NotificationServer
	logger             *slog.Logger
	timeout            time.Duration
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:               addr,
		notificationServer: NewNotificationServer(service, logger),
		logger:             logger,
		timeout:            timeout,
	}
}

func (s *GRPCServer) Run() error {
	return grpc_server.NewServer(
		s.addr,
		s.logger,
		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				notificationv1.RegisterNotificationServiceServer(r, s.notificationServer)
			},
		),
		grpc_server.WithUnaryInterceptors(
			interceptors.Validate(),
		),
		grpc_server.WithHandlerTimeout(s.timeout),
	)
}
