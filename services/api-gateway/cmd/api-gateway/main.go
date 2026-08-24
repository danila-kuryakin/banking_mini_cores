package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/httpapi"

	gwconfig "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "api-gateway: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func run() error {
	cfg, err := gwconfig.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = httpapi.New(cfg, log)
	if err != nil {
		return fmt.Errorf("build gateway: %w", err)
	}

	return nil
}
