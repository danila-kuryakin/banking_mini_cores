package password

import (
	"context"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
)

var _ interfaces.PasswordHasher = (*Hasher)(nil)

// Hasher ограничивает число одновременных вычислений argon2: каждое занимает ARGON2_MEMORY.
type Hasher struct {
	slots   chan struct{}
	maxWait time.Duration
}

func NewHasher(maxConcurrent int, maxWait time.Duration) *Hasher {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	return &Hasher{
		slots:   make(chan struct{}, maxConcurrent),
		maxWait: maxWait,
	}
}

func (h *Hasher) Hash(ctx context.Context, plain string) (string, error) {
	release, err := h.acquire(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	return Hash(plain)
}

func (h *Hasher) Verify(ctx context.Context, encoded, plain string) error {
	release, err := h.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	return Verify(encoded, plain)
}

// acquire ждёт свободное место не дольше maxWait и не дольше ctx.
func (h *Hasher) acquire(ctx context.Context) (func(), error) {
	// Отменённый контекст проверяется до select: при свободном месте select выбрал бы ветку случайно.
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrPasswordHasherBusy, err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, h.maxWait)
	defer cancel()

	select {
	case h.slots <- struct{}{}:
		return func() { <-h.slots }, nil
	case <-waitCtx.Done():
		return nil, fmt.Errorf("%w: %w", domain.ErrPasswordHasherBusy, waitCtx.Err())
	}
}
