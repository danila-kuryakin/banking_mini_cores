package worker

import (
	"context"
	"time"
)

// backoff удваивает паузу от min до max.
type backoff struct {
	min     time.Duration
	max     time.Duration
	current time.Duration
}

func newBackoff(min, max time.Duration) *backoff {
	return &backoff{min: min, max: max}
}

// wait спит очередную паузу; false - контекст отменён, повторять незачем.
func (b *backoff) wait(ctx context.Context) bool {
	if b.current == 0 {
		b.current = b.min
	} else {
		b.current = min(b.current*2, b.max)
	}

	timer := time.NewTimer(b.current)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (b *backoff) reset() {
	b.current = 0
}
