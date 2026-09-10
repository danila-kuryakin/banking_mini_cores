package server

import (
	"context"
	"fmt"
	"log/slog"

	grpcSrv "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/grpc/server"
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

func NewServer(cfg *config.Config, service *service.Service, jwks func() token.Set, logger *slog.Logger) *Server {
	grpcServer := grpcSrv.NewGRPCServer(cfg.Server.GetAddr(), service, logger, cfg.RequestTimeout)

	jwksServer := httpSrv.NewJWKSServer(cfg.JWKS.GetAddr(), jwks, logger)

	return &Server{
		grpc:   grpcServer,
		jwks:   jwksServer,
		logger: logger,
	}
}

func (s *Server) Run() error {
	errCh := make(chan error, 2)

	go func() {
		s.logger.Info("jwks listening", "addr", s.jwks.Addr(), "path", domain.JWKS_PATH)

		if err := s.jwks.Run(); err != nil {
			errCh <- fmt.Errorf("jwks server: %w", err)

			return
		}

		errCh <- nil
	}()

	go func() {
		if err := s.grpc.Run(); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)

			return
		}

		errCh <- nil
	}()

	defer s.shutdownJWKS()

	return <-errCh
}

func (s *Server) shutdownJWKS() {
	ctx, cancel := context.WithTimeout(context.Background(), domain.DEFAULT_SHUTDOWN_TIMEOUT)
	defer cancel()

	if err := s.jwks.Shutdown(ctx); err != nil {
		s.logger.Error("failed to shutdown jwks server", "error", err)
	}
}
