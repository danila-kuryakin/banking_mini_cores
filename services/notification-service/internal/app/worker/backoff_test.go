package worker

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repeatDuration возвращает n одинаковых пауз
func repeatDuration(d time.Duration, n int) []time.Duration {
	out := make([]time.Duration, n)
	for i := range out {
		out[i] = d
	}
	return out
}

// measureWaits делает n вызовов wait при живом ctx и возвращает длительность каждого
func measureWaits(t *testing.T, b *backoff, n int) []time.Duration {
	t.Helper()
	out := make([]time.Duration, 0, n)
	for range n {
		start := time.Now()
		require.True(t, b.wait(t.Context()))
		out = append(out, time.Since(start))
	}
	return out
}

func TestBackoff_Wait(t *testing.T) {
	tests := []struct {
		name  string
		min   time.Duration
		max   time.Duration
		waits int
		want  []time.Duration
	}{
		{
			name:  "doubling up to max",
			min:   500 * time.Millisecond,
			max:   4 * time.Second,
			waits: 5,
			want:  []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second},
		},
		{
			name:  "min equals max",
			min:   time.Second,
			max:   time.Second,
			waits: 3,
			want:  []time.Duration{time.Second, time.Second, time.Second},
		},
		{
			name:  "max is not power of two of min",
			min:   time.Second,
			max:   3 * time.Second,
			waits: 4,
			// 2s удваивается до 4s и срезается до 3s
			want: []time.Duration{time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second},
		},
		{
			name:  "first wait is min",
			min:   700 * time.Millisecond,
			max:   10 * time.Second,
			waits: 1,
			want:  []time.Duration{700 * time.Millisecond},
		},
		{
			name:  "long growth does not overflow",
			min:   time.Second,
			max:   time.Hour,
			waits: 70,
			// 1s..2048s (12 пауз), 4096s > 3600s - дальше 58 пауз по 1h
			want: append([]time.Duration{
				time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
				16 * time.Second, 32 * time.Second, 64 * time.Second, 128 * time.Second,
				256 * time.Second, 512 * time.Second, 1024 * time.Second, 2048 * time.Second,
			}, repeatDuration(time.Hour, 58)...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := newBackoff(tt.min, tt.max)

				got := measureWaits(t, b, tt.waits)

				assert.Equal(t, tt.want, got)
			})
		})
	}
}

func TestBackoff_Reset(t *testing.T) {
	tests := []struct {
		name        string
		min         time.Duration
		max         time.Duration
		waitsBefore int
		resets      int
		waitsAfter  int
		want        []time.Duration
	}{
		{
			name:        "reset in the middle of growth",
			min:         500 * time.Millisecond,
			max:         4 * time.Second,
			waitsBefore: 3,
			resets:      1,
			waitsAfter:  2,
			want:        []time.Duration{500 * time.Millisecond, time.Second},
		},
		{
			name:        "reset after reaching max",
			min:         500 * time.Millisecond,
			max:         4 * time.Second,
			waitsBefore: 5,
			resets:      1,
			waitsAfter:  1,
			want:        []time.Duration{500 * time.Millisecond},
		},
		{
			name:        "reset on fresh backoff",
			min:         500 * time.Millisecond,
			max:         4 * time.Second,
			waitsBefore: 0,
			resets:      1,
			waitsAfter:  2,
			want:        []time.Duration{500 * time.Millisecond, time.Second},
		},
		{
			name:        "double reset",
			min:         500 * time.Millisecond,
			max:         4 * time.Second,
			waitsBefore: 3,
			resets:      2,
			waitsAfter:  1,
			want:        []time.Duration{500 * time.Millisecond},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := newBackoff(tt.min, tt.max)
				measureWaits(t, b, tt.waitsBefore)
				for range tt.resets {
					b.reset()
				}

				got := measureWaits(t, b, tt.waitsAfter)

				assert.Equal(t, tt.want, got)
			})
		})
	}
}

func TestBackoff_WaitCanceled(t *testing.T) {
	tests := []struct {
		name        string
		min         time.Duration
		max         time.Duration
		waitsBefore int
		makeCtx     func(parent context.Context) (context.Context, context.CancelFunc)
		wantElapsed time.Duration
	}{
		{
			name: "canceled before wait",
			min:  time.Second,
			max:  4 * time.Second,
			makeCtx: func(parent context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(parent)
				cancel()
				return ctx, cancel
			},
			wantElapsed: 0,
		},
		{
			name: "canceled during first pause",
			min:  time.Second,
			max:  4 * time.Second,
			makeCtx: func(parent context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(parent)
				time.AfterFunc(300*time.Millisecond, cancel)
				return ctx, cancel
			},
			wantElapsed: 300 * time.Millisecond,
		},
		{
			name:        "canceled during third pause",
			min:         time.Second,
			max:         4 * time.Second,
			waitsBefore: 2,
			makeCtx: func(parent context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(parent)
				// паузы 1s и 2s, отмена через 700ms после начала третьей
				time.AfterFunc(3700*time.Millisecond, cancel)
				return ctx, cancel
			},
			wantElapsed: 700 * time.Millisecond,
		},
		{
			name: "deadline shorter than pause",
			min:  time.Second,
			max:  4 * time.Second,
			makeCtx: func(parent context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(parent, 200*time.Millisecond)
			},
			wantElapsed: 200 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := newBackoff(tt.min, tt.max)
				ctx, cancel := tt.makeCtx(t.Context())
				defer cancel()
				for range tt.waitsBefore {
					require.True(t, b.wait(ctx))
				}

				start := time.Now()
				ok := b.wait(ctx)
				elapsed := time.Since(start)

				assert.False(t, ok)
				assert.Equal(t, tt.wantElapsed, elapsed)
			})
		})
	}
}

func TestBackoff_InvariantBounded(t *testing.T) {
	tests := []struct {
		name  string
		min   time.Duration
		max   time.Duration
		waits int
	}{
		{name: "500ms to 4s", min: 500 * time.Millisecond, max: 4 * time.Second, waits: 10},
		{name: "min equals max", min: time.Second, max: time.Second, waits: 5},
		{name: "1ns to 1h", min: time.Nanosecond, max: time.Hour, waits: 100},
		{name: "3ms to 10ms", min: 3 * time.Millisecond, max: 10 * time.Millisecond, waits: 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := newBackoff(tt.min, tt.max)

				got := measureWaits(t, b, tt.waits)

				prev := tt.min
				for i, pause := range got {
					assert.GreaterOrEqualf(t, pause, tt.min, "pause %d below min", i)
					assert.LessOrEqualf(t, pause, tt.max, "pause %d above max", i)
					assert.GreaterOrEqualf(t, pause, prev, "pause %d decreased", i)
					prev = pause
				}
			})
		})
	}
}
