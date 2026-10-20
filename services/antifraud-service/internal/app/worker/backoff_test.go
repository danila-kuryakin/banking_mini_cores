package worker

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_OP_WAIT  = "wait"
	TEST_OP_RESET = "reset"
)

// runBackoffOps - выполняет wait/reset по списку и возвращает длительность и результат каждого wait
func runBackoffOps(ctx context.Context, b *backoff, ops []string) ([]time.Duration, []bool) {
	var durations []time.Duration
	var results []bool
	for _, op := range ops {
		if op == TEST_OP_RESET {
			b.reset()
			continue
		}
		start := time.Now()
		results = append(results, b.wait(ctx))
		durations = append(durations, time.Since(start))
	}
	return durations, results
}

// waits - n операций wait подряд
func waits(n int) []string {
	ops := make([]string, 0, n)
	for range n {
		ops = append(ops, TEST_OP_WAIT)
	}
	return ops
}

// trues - n значений true
func trues(n int) []bool {
	out := make([]bool, 0, n)
	for range n {
		out = append(out, true)
	}
	return out
}

func TestBackoff_Wait(t *testing.T) {
	tests := []struct {
		name          string
		minBackoff    time.Duration
		maxBackoff    time.Duration
		ops           []string
		wantDurations []time.Duration
	}{
		{
			name:       "doubles from min up to max",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 30 * time.Second,
			ops:        waits(8),
			wantDurations: []time.Duration{
				500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second,
				8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second,
			},
		},
		{
			name:          "min equals max",
			minBackoff:    time.Second,
			maxBackoff:    time.Second,
			ops:           waits(3),
			wantDurations: []time.Duration{time.Second, time.Second, time.Second},
		},
		{
			name:       "max not multiple of min",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 3 * time.Second,
			ops:        waits(5),
			wantDurations: []time.Duration{
				500 * time.Millisecond, time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second,
			},
		},
		{
			name:          "reset returns to min",
			minBackoff:    500 * time.Millisecond,
			maxBackoff:    30 * time.Second,
			ops:           []string{TEST_OP_WAIT, TEST_OP_WAIT, TEST_OP_RESET, TEST_OP_WAIT},
			wantDurations: []time.Duration{500 * time.Millisecond, time.Second, 500 * time.Millisecond},
		},
		{
			name:          "reset of fresh backoff",
			minBackoff:    500 * time.Millisecond,
			maxBackoff:    30 * time.Second,
			ops:           []string{TEST_OP_RESET, TEST_OP_WAIT},
			wantDurations: []time.Duration{500 * time.Millisecond},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := newBackoff(tt.minBackoff, tt.maxBackoff)

				durations, results := runBackoffOps(t.Context(), b, tt.ops)

				assert.Equal(t, tt.wantDurations, durations)
				assert.Equal(t, trues(len(tt.wantDurations)), results)
			})
		})
	}
}

func TestBackoff_Wait_Cancelled(t *testing.T) {
	tests := []struct {
		name        string
		warmup      int
		cancelAfter time.Duration
	}{
		{name: "cancelled during first pause", warmup: 0, cancelAfter: 200 * time.Millisecond},
		{name: "already cancelled context", warmup: 0, cancelAfter: 0},
		// после пауз 0.5s, 1s, 2s идёт пауза 4s, отмена через 1s
		{name: "cancelled during fourth pause", warmup: 3, cancelAfter: time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := newBackoff(500*time.Millisecond, 30*time.Second)
				for range tt.warmup {
					require.True(t, b.wait(t.Context()))
				}
				ctx, cancel := context.WithTimeout(t.Context(), tt.cancelAfter)
				defer cancel()

				start := time.Now()
				ok := b.wait(ctx)
				elapsed := time.Since(start)

				assert.False(t, ok)
				assert.Equal(t, tt.cancelAfter, elapsed)
			})
		})
	}
}
