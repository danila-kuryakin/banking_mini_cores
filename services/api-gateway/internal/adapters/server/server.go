package server

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/grpc"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
)

type Server struct {
	rest            *rest.RestServer
	health          *grpc.HealthServer
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func NewServer(cfg *config.Config, handler *handler.Handler, logger *slog.Logger) *Server {
	return &Server{
		rest:            rest.NewRestServer(cfg.RestServer.GetAddr(), handler, logger),
		health:          grpc.NewHealthServer(cfg.GRPCServer.GetAddr()),
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

func (s *Server) Run() error {
	s.rest.Init()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		if err := s.health.Run(); err != nil {
			errCh <- fmt.Errorf("health server: %w", err)
		}
	}()

	go func() {
		if err := s.rest.Run(); err != nil {
			errCh <- fmt.Errorf("rest server: %w", err)
		}
	}()

	s.logger.Info("api-gateway listening",
		"http", s.rest.Addr(),
		"health", s.health.Addr(),
		"swagger", s.rest.SwaggerURL(),
	)

	select {
	case err := <-errCh:
		err = s.shutdown()
		if err != nil {
			return err
		}
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	s.logger.Info("shutting down")

	s.health.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.rest.Shutdown(ctx); err != nil {
		return fmt.Errorf("rest shutdown: %w", err)
	}

	s.logger.Info("stopped gracefully")

	return nil
}

func (s *Server) SetServingStatus(service string, serving bool) {
	s.health.SetServingStatus(service, serving)
}
