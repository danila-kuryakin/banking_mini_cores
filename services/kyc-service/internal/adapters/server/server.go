package server

import (
	"log/slog"

	grpcSrv "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/config"
)

type Server struct {
	grpc   *grpcSrv.GRPCServer
	logger *slog.Logger
}

func NewServer(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	grpcServer := grpcSrv.NewGRPCServer(cfg.Server.GetAddr(), logger, cfg.RequestTimeout)

	return &Server{
		grpc:   grpcServer,
		logger: logger,
	}, nil
}

func (s *Server) Run() error {
	return s.grpc.Run()
}
