package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/grpcapi"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/config"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "customer-service: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	return grpcapi.NewServer(cfg, log)
}
