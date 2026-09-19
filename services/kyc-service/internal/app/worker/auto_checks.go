package worker

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/app/checks"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// AutoChecks - фоновый разбор поданных заявок.
//
// Шаг асинхронный: SubmitApplication не ждёт результата проверок,
// клиент получает ответ сразу, а заявка уезжает в очередь воркера.
type AutoChecks struct {
	repo      *repository.Repository
	clients   *client.Clients
	runner    *checks.Runner
	log       *slog.Logger
	interval  time.Duration
	batchSize int
}

func NewAutoChecks(
	repo *repository.Repository,
	clients *client.Clients,
	log *slog.Logger,
	interval time.Duration,
	batchSize int,
) *AutoChecks {
	return &AutoChecks{
		repo:      repo,
		clients:   clients,
		runner:    checks.NewRunner(repo.Sanctions),
		log:       log,
		interval:  interval,
		batchSize: batchSize,
	}
}

// Run крутится до отмены контекста. Ошибка разбора одной заявки не роняет
// воркер: заявка остаётся в auto_checks и будет разобрана после перезапуска.
func (w *AutoChecks) Run(ctx context.Context) {
	w.log.Info("auto checks worker started", "interval", w.interval.String(), "batch_size", w.batchSize)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("auto checks worker stopped")

			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *AutoChecks) processBatch(ctx context.Context) {
	taken, err := w.repo.Application.TakeForAutoChecks(ctx, w.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			w.log.Error("failed to take applications", "error", err)
		}

		return
	}

	for _, app := range taken {
		if err := w.process(ctx, app); err != nil {
			w.log.Error("failed to process application", "application_id", app.ID.String(), "error", err)
		}
	}
}

func (w *AutoChecks) process(ctx context.Context, app *models.Application) error {
	results, err := w.runner.Run(ctx, app)
	if err != nil {
		return err
	}

	failed := make([]string, 0, len(results))

	for _, check := range results {
		if _, err := w.repo.Check.SaveCheck(ctx, check); err != nil {
			return err
		}

		if !check.Passed() {
			failed = append(failed, string(check.Type)+": "+check.Details)
		}
	}

	if len(failed) == 0 {
		_, err = w.repo.Application.Transition(ctx, app.ID,
			models.Transition{To: models.STATUS_IN_REVIEW},
			models.AllowedFrom(models.STATUS_IN_REVIEW),
		)
		if err != nil {
			return err
		}

		w.log.Info("application passed auto checks", "application_id", app.ID.String())

		return nil
	}

	reason := domain.REASON_AUTO_REJECTED + ": " + strings.Join(failed, "; ")

	_, err = w.repo.Application.Transition(ctx, app.ID, models.Transition{
		To:      models.STATUS_AUTO_REJECTED,
		Comment: reason,
		Reason:  reason,
		Decided: true,
	}, models.AllowedFrom(models.STATUS_AUTO_REJECTED))
	if err != nil {
		return err
	}

	err = w.clients.Customer.SetStatus(ctx, app.UserID, models.CUSTOMER_STATUS_PROFILE_FILLED, reason, nil)
	if err != nil {
		return err
	}

	w.log.Info("application auto rejected", "application_id", app.ID.String(), "reason", reason)

	return nil
}
