package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// DailyTotals копит дневные суммы переводов по событиям ledger.transaction_completed.
type DailyTotals struct {
	reader     interfaces.EventReader
	totals     interfaces.TotalsRepository
	log        *slog.Logger
	minBackoff time.Duration
	maxBackoff time.Duration
}

func NewDailyTotals(
	reader interfaces.EventReader,
	totals interfaces.TotalsRepository,
	log *slog.Logger,
	minBackoff time.Duration,
	maxBackoff time.Duration,
) *DailyTotals {
	return &DailyTotals{
		reader:     reader,
		totals:     totals,
		log:        log,
		minBackoff: minBackoff,
		maxBackoff: maxBackoff,
	}
}

// Run читает до отмены контекста; оффсет коммитится только после записи в базу, поэтому доставка at-least-once.
func (w *DailyTotals) Run(ctx context.Context) {
	w.log.Info("daily totals consumer started")

	pause := newBackoff(w.minBackoff, w.maxBackoff)

	for {
		msg, err := w.reader.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				w.log.Info("daily totals consumer stopped")

				return
			}

			w.log.Error("failed to fetch event", "error", err)

			if !pause.wait(ctx) {
				return
			}

			continue
		}

		if !w.process(ctx, msg) {
			return
		}

		for {
			if err := w.reader.Commit(ctx, msg); err == nil {
				break
			} else if ctx.Err() == nil {
				w.log.Error("failed to commit offset", "offset", msg.Offset, "error", err)
			}

			if !pause.wait(ctx) {
				return
			}
		}

		pause.reset()
	}
}

// process повторяет запись до успеха; битое сообщение пропускается, false - контекст отменён.
func (w *DailyTotals) process(ctx context.Context, msg models.Message) bool {
	pause := newBackoff(w.minBackoff, w.maxBackoff)

	for {
		err := w.handle(ctx, msg)
		if err == nil {
			return true
		}

		if errors.Is(err, domain.ErrPoisonMessage) {
			w.log.Error("skipping poison event", "partition", msg.Partition, "offset", msg.Offset, "error", err)

			return true
		}

		w.log.Error("failed to handle event, will retry", "offset", msg.Offset, "error", err)

		if !pause.wait(ctx) {
			return false
		}
	}
}

func (w *DailyTotals) handle(ctx context.Context, msg models.Message) error {
	envelope, err := models.ParseEnvelope(msg.Value)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrPoisonMessage, err)
	}

	if envelope.EventType != models.EVENT_TRANSACTION_COMPLETED {
		return nil
	}

	var payload models.TransactionPayload

	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("%w: payload: %v", domain.ErrPoisonMessage, err)
	}

	if payload.Type != models.TRANSACTION_TYPE_TRANSFER {
		return nil
	}

	if payload.FromAccountID == uuid.Nil {
		return fmt.Errorf("%w: payload: from_account_id is required", domain.ErrPoisonMessage)
	}

	if payload.Amount <= 0 {
		return fmt.Errorf("%w: payload: amount must be positive", domain.ErrPoisonMessage)
	}

	if payload.CreatedAt.IsZero() {
		return fmt.Errorf("%w: payload: created_at is required", domain.ErrPoisonMessage)
	}

	currency, err := models.ParseCurrency(payload.Currency)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrPoisonMessage, err)
	}

	added, err := w.totals.AddTransfer(ctx, models.CompletedTransfer{
		EventID:   envelope.EventID,
		EventType: envelope.EventType,
		AccountID: payload.FromAccountID,
		Currency:  currency,
		Amount:    payload.Amount,
		Day:       models.Day(payload.CreatedAt),
	})
	if err != nil {
		return err
	}

	if !added {
		w.log.Debug("duplicate event skipped", "event_id", envelope.EventID)
	}

	return nil
}
