package server

import (
	"log/slog"

	grpcSrv "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/adapters/grpc/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/config"
)

type Server struct {
	grpc   *grpcSrv.GRPCServer
	logger *slog.Logger
}

func NewServer(cfg *config.Config, service *service.Service, logger *slog.Logger) (*Server, error) {
	grpcServer := grpcSrv.NewGRPCServer(cfg.Server.GetAddr(), service, logger, cfg.RequestTimeout)

	return &Server{
		grpc:   grpcServer,
		logger: logger,
	}, nil
}

func (s *Server) Run() error {
	return s.grpc.Run()
}
