package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// HandleFunc обрабатывает одно сообщение; ErrPoisonMessage - повторять бесполезно.
type HandleFunc func(ctx context.Context, msg models.Message) error

// Consumer: обработка с повторами, после них - DLQ, и только потом коммит оффсета.
type Consumer struct {
	reader      interfaces.EventReader
	deadLetter  interfaces.DeadLetterWriter
	handle      HandleFunc
	log         *slog.Logger
	maxAttempts int
	minBackoff  time.Duration
	maxBackoff  time.Duration
}

func NewConsumer(
	reader interfaces.EventReader,
	deadLetter interfaces.DeadLetterWriter,
	handle HandleFunc,
	log *slog.Logger,
	maxAttempts int,
	minBackoff time.Duration,
	maxBackoff time.Duration,
) *Consumer {
	return &Consumer{
		reader:      reader,
		deadLetter:  deadLetter,
		handle:      handle,
		log:         log,
		maxAttempts: maxAttempts,
		minBackoff:  minBackoff,
		maxBackoff:  maxBackoff,
	}
}

// Run читает до отмены контекста; оффсет двигается, только когда сообщение обработано или лежит в DLQ.
func (c *Consumer) Run(ctx context.Context) {
	c.log.Info("consumer started", "max_attempts", c.maxAttempts)

	pause := newBackoff(c.minBackoff, c.maxBackoff)

	for {
		msg, err := c.reader.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("consumer stopped")

				return
			}

			c.log.Error("failed to fetch event", "error", err)

			if !pause.wait(ctx) {
				return
			}

			continue
		}

		if !c.process(ctx, msg) {
			return
		}

		if !c.commit(ctx, msg) {
			return
		}

		pause.reset()
	}
}

// process: false - контекст отменён, сообщение не обработано и придёт снова.
func (c *Consumer) process(ctx context.Context, msg models.Message) bool {
	pause := newBackoff(c.minBackoff, c.maxBackoff)

	for attempt := 1; ; attempt++ {
		err := c.handle(ctx, msg)
		if err == nil {
			return true
		}

		if ctx.Err() != nil {
			return false
		}

		if errors.Is(err, domain.ErrPoisonMessage) {
			return c.park(ctx, msg, err)
		}

		if attempt >= c.maxAttempts {
			return c.park(ctx, msg, fmt.Errorf("gave up after %d attempts: %w", attempt, err))
		}

		c.log.Warn("failed to handle event, will retry", "attempt", attempt, "offset", msg.Offset, "error", err)

		if !pause.wait(ctx) {
			return false
		}
	}
}

// park кладёт сообщение в DLQ; без этого оффсет двигать нельзя, поэтому запись повторяется до успеха.
func (c *Consumer) park(ctx context.Context, msg models.Message, reason error) bool {
	pause := newBackoff(c.minBackoff, c.maxBackoff)

	for {
		err := c.deadLetter.Send(ctx, msg, reason)
		if err == nil {
			c.log.Error("event moved to dlq", "partition", msg.Partition, "offset", msg.Offset, "reason", reason)

			return true
		}

		if ctx.Err() == nil {
			c.log.Error("failed to write dlq", "offset", msg.Offset, "error", err)
		}

		if !pause.wait(ctx) {
			return false
		}
	}
}

func (c *Consumer) commit(ctx context.Context, msg models.Message) bool {
	pause := newBackoff(c.minBackoff, c.maxBackoff)

	for {
		err := c.reader.Commit(ctx, msg)
		if err == nil {
			return true
		}

		if ctx.Err() == nil {
			c.log.Error("failed to commit offset", "offset", msg.Offset, "error", err)
		}

		if !pause.wait(ctx) {
			return false
		}
	}
}
