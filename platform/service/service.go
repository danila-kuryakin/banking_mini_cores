package service

import (
	"context"
	"log/slog"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/platform/logging"
	"github.com/danila-kuryakin/banking_mini_cores/platform/observability"
	"github.com/danila-kuryakin/banking_mini_cores/platform/runner"
)

type App struct {
	Base   config.Base
	Health *observability.Health
	Runner *runner.Runner
}

func New(base config.Base) (*App, error) {
	logger := logging.New(base.ServiceName, base.LogLevel, base.IsDev())

	logger.Info("starting service",
		slog.String("env", base.Env),
		slog.String("grpc_addr", base.GRPCAddr),
	)

	app := &App{
		Base:    base,
		Logger:  logger,
		Metrics: observability.NewMetrics(base.ServiceName),
		Health:  observability.NewHealth(),
		Runner:  runner.New(logger, base.ShutdownTimeout),
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	a.Runner.BeforeStop(a.Health.BeginShutdown)

	err := a.Runner.Run(ctx)

	return err
}
