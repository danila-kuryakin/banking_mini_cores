package grpc

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/interceptors"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/app/service"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

type GRPCServer struct {
	addr       string
	fileServer *FileServer
	logger     *slog.Logger
	timeout    time.Duration
}

func NewGRPCServer(addr string, service *service.Service, logger *slog.Logger, timeout time.Duration) *GRPCServer {
	return &GRPCServer{
		addr:       addr,
		fileServer: NewFileServer(service, logger),
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
				filestorev1.RegisterFileServiceServer(r, s.fileServer)
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
