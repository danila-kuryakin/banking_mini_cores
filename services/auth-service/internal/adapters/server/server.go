package server

import (
	"context"
	"log/slog"

	grpcSrv "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/grpc"
	httpSrv "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/http"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

type Server struct {
	grpc   *grpcSrv.GRPCServer
	jwks   *httpSrv.JWKSServer
	logger *slog.Logger
}

func NewServer(cfg *config.Config, service *service.Service, keys token.Set, logger *slog.Logger) (*Server, error) {
	grpcServer := grpcSrv.NewGRPCServer(cfg.Server.GetAddr(), service, logger)

	jwksServer, err := httpSrv.NewJWKSServer(cfg.JWKS.GetAddr(), keys)
	if err != nil {
		return nil, err
	}

	return &Server{
		grpc:   grpcServer,
		jwks:   jwksServer,
		logger: logger,
	}, nil
}

func (s *Server) Run() error {
	go func() {
		s.logger.Info("jwks listening", "addr", s.jwks.Addr(), "path", domain.JWKS_PATH)

		if err := s.jwks.Run(); err != nil {
			s.logger.Error("auth server stopped", "error", err)
		}
	}()

	defer s.shutdownJWKS()

	return s.grpc.Run()
}

func (s *Server) shutdownJWKS() {
	ctx, cancel := context.WithTimeout(context.Background(), domain.DEFAULT_SHUTDOWN_TIMEOUT)
	defer cancel()

	if err := s.jwks.Shutdown(ctx); err != nil {
		s.logger.Error("failed to shutdown auth server", "error", err)
	}
}
