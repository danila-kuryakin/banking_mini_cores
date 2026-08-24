package service

import (
	"context"
	"fmt"
	"os"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

type Setup func(ctx context.Context, app *App) error

func Start(serviceName string, defaultGRPCPort int, setup Setup) {
	if err := run(serviceName, defaultGRPCPort, setup); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "%s: %v\n", serviceName, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func run(serviceName string, defaultGRPCPort int, setup Setup) error {
	ctx := context.Background()

	base, err := config.LoadBase(serviceName, defaultGRPCPort)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	app, err := New(base)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	if setup != nil {
		if err := setup(ctx, app); err != nil {
			return fmt.Errorf("setup: %w", err)
		}
	}

	return app.Run(ctx)
}
