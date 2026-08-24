// Package runner owns the process lifecycle: start the components, wait for a
// termination signal, and shut everything down in the right order within a
// bounded time.
//
// Getting this wrong is how a deploy drops requests, so it is written once
// here instead of nine times in nine main functions.
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// Component is one long-running part of a service: the gRPC server, the admin
// HTTP server, a Kafka consumer, a background worker.
type Component struct {
	// Name appears in the startup and shutdown logs.
	Name string

	// Run blocks until the component stops. Returning nil means a clean stop;
	// any other error brings the whole process down, because a service
	// running with half its components is worse than one that restarts.
	Run func(ctx context.Context) error

	// Stop asks the component to finish. It is called with a context bounded
	// by the shutdown timeout.
	Stop func(ctx context.Context) error
}

// Runner supervises a set of components.
type Runner struct {
	logger     *slog.Logger
	timeout    time.Duration
	components []Component

	// beforeStop runs before any component is stopped. It is where readiness
	// is flipped to failing, so traffic drains before the listeners close.
	beforeStop []func()
}

// New creates a Runner. timeout bounds the whole shutdown sequence.
func New(logger *slog.Logger, timeout time.Duration) *Runner {
	return &Runner{logger: logger, timeout: timeout}
}

// Add registers a component. Components start concurrently and stop in
// reverse registration order, so a dependency registered first is torn down
// last.
func (r *Runner) Add(c Component) { r.components = append(r.components, c) }

// BeforeStop registers a hook that runs at the very start of shutdown.
func (r *Runner) BeforeStop(fn func()) { r.beforeStop = append(r.beforeStop, fn) }

// Run starts every component and blocks until SIGINT or SIGTERM arrives, or
// until a component fails. It then stops everything and returns the first
// error that was not a clean shutdown.
func (r *Runner) Run(ctx context.Context) error {
	ctx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	group, groupCtx := errgroup.WithContext(ctx)

	for _, component := range r.components {
		r.logger.Info("starting component", slog.String("component", component.Name))

		group.Go(func() error {
			if err := component.Run(groupCtx); err != nil {
				return fmt.Errorf("component %s: %w", component.Name, err)
			}

			return nil
		})
	}

	// Wait for a signal, a cancelled parent, or a component failure.
	<-groupCtx.Done()

	r.logger.Info("shutdown initiated", slog.Duration("timeout", r.timeout))

	for _, hook := range r.beforeStop {
		hook()
	}

	// A fresh context: the one above is already cancelled, and shutdown work
	// needs its own budget.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.timeout)
	defer cancel()

	for i := len(r.components) - 1; i >= 0; i-- {
		component := r.components[i]
		if component.Stop == nil {
			continue
		}

		r.logger.Info("stopping component", slog.String("component", component.Name))

		if err := component.Stop(shutdownCtx); err != nil {
			r.logger.Error("component did not stop cleanly",
				slog.String("component", component.Name),
				slog.String("error", err.Error()),
			)
		}
	}

	err := group.Wait()

	// A cancelled context is the expected outcome of a signal, not a failure.
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	r.logger.Info("shutdown complete")

	return nil
}
