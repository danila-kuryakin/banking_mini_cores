// Command customer-service runs the customer-service gRPC server.
//
// Responsibility: customer profiles and the customer lifecycle status.
//
// The body is deliberately this short. Everything about how a service starts,
// serves, observes itself and shuts down lives in platform/service; what
// belongs here is only what makes this service this service.
package main

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/platform/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/grpcapi"
)

// defaultGRPCPort is used when GRPC_ADDR is unset, so `go run ./cmd/customer-service`
// works with no environment at all.
const defaultGRPCPort = 50052

func main() {
	service.Main("customer-service", defaultGRPCPort, func(_ context.Context, app *service.App) error {
		api := grpcapi.NewServer()

		opts := app.GRPCOptions()
		opts.Policy = grpcapi.Policy()
		opts.Register = api.RegisterOn

		app.AddGRPC(opts)

		return nil
	})
}
