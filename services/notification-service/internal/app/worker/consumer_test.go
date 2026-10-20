package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// Параметры Consumer по умолчанию
const (
	TEST_MAX_ATTEMPTS = 3
	TEST_MIN_BACKOFF  = 500 * time.Millisecond
	TEST_MAX_BACKOFF  = 4 * time.Second
)

var (
	errTestHandle = errors.New("handle failed")
	errTestSend   = errors.New("dlq send failed")
	errTestCommit = errors.New("commit failed")
	errTestFetch  = errors.New("broker unavailable")
)

// consumerSetup - сценарий фейков и параметры Consumer для кейса
type consumerSetup struct {
	maxAttempts     int
	minBackoff      time.Duration
	maxBackoff      time.Duration
	fetches         []fetchResult
	fetchAlwaysErr  error
	handleErrs      map[int64][]error
	sendErrs        []error
	sendAlwaysErr   error
	commitErrs      []error
	commitAlwaysErr error
	stopAt          int
}

type consumerFixture struct {
	log      *callLog
	reader   *fakeEventReader
	dlq      *fakeDeadLetterWriter
	handler  *fakeHandler
	consumer *Consumer
}

func newConsumerFixture(s consumerSetup) *consumerFixture {
	log := newCallLog(s.stopAt)
	reader := newFakeEventReader(log, s.fetches, s.fetchAlwaysErr, s.commitErrs, s.commitAlwaysErr)
	dlq := newFakeDeadLetterWriter(log, s.sendErrs, s.sendAlwaysErr)
	handler := newFakeHandler(log, s.handleErrs)
	consumer := NewConsumer(reader, dlq, handler.handle, slog.New(slog.DiscardHandler),
		s.maxAttempts, s.minBackoff, s.maxBackoff)
	return &consumerFixture{log: log, reader: reader, dlq: dlq, handler: handler, consumer: consumer}
}

// runConsumer запускает Run в горутине; канал закрывается, когда Run вернулся
func runConsumer(ctx context.Context, c *Consumer) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		c.Run(ctx)
		close(done)
	}()
	return done
}

// withoutFetches оставляет в журнале только handle, send и commit
func withoutFetches(calls []call) []call {
	var out []call
	for _, c := range calls {
		if c.Op != TEST_OP_FETCH && c.Op != TEST_OP_FETCH_WAIT {
			out = append(out, c)
		}
	}
	return out
}

func fetched(msg models.Message, at time.Duration) call {
	return call{Op: TEST_OP_FETCH, Msg: msg, At: at}
}

func fetchFailed(err error, at time.Duration) call {
	return call{Op: TEST_OP_FETCH, Err: err, At: at}
}

func fetchWait(at time.Duration) call {
	return call{Op: TEST_OP_FETCH_WAIT, At: at}
}

func handled(msg models.Message, err error, at time.Duration) call {
	return call{Op: TEST_OP_HANDLE, Msg: msg, Err: err, At: at}
}

func sent(msg models.Message, err error, at time.Duration) call {
	return call{Op: TEST_OP_SEND, Msg: msg, Err: err, At: at}
}

func committed(msg models.Message, err error, at time.Duration) call {
	return call{Op: TEST_OP_COMMIT, Msg: msg, Err: err, At: at}
}

// defaultSetup - параметры по умолчанию: 3 попытки, 500ms..4s
func defaultSetup() consumerSetup {
	return consumerSetup{maxAttempts: TEST_MAX_ATTEMPTS, minBackoff: TEST_MIN_BACKOFF, maxBackoff: TEST_MAX_BACKOFF}
}

func TestConsumer_Run(t *testing.T) {
	const ms = time.Millisecond
	m1, m2, m3 := testMessage(1), testMessage(2), testMessage(3)
	errWrapped := fmt.Errorf("decode payload: %w", errTestHandle)
	errPoisonWrapped := fmt.Errorf("bad json: %w", domain.ErrPoisonMessage)
	poison := domain.ErrPoisonMessage

	with := func(mod func(s *consumerSetup)) consumerSetup {
		s := defaultSetup()
		mod(&s)
		return s
	}

	tests := []struct {
		name        string
		setup       consumerSetup
		want        []call
		wantReasons [][]error
	}{
		{
			name: "single message success",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
			}),
			want: []call{fetched(m1, 0), handled(m1, nil, 0), committed(m1, nil, 0), fetchWait(0)},
		},
		{
			name: "three messages in fetch order",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}, {msg: m3}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, nil, 0), committed(m1, nil, 0),
				fetched(m2, 0), handled(m2, nil, 0), committed(m2, nil, 0),
				fetched(m3, 0), handled(m3, nil, 0), committed(m3, nil, 0),
				fetchWait(0),
			},
		},
		{
			name: "success on third attempt",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle, errTestHandle}}
			}),
			// паузы 500ms и 1s: попытки в 0, 500ms, 1.5s
			want: []call{
				fetched(m1, 0),
				handled(m1, errTestHandle, 0), handled(m1, errTestHandle, 500*ms), handled(m1, nil, 1500*ms),
				committed(m1, nil, 1500*ms), fetchWait(1500 * ms),
			},
		},
		{
			name: "success on second attempt",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, errTestHandle, 0), handled(m1, nil, 500*ms),
				committed(m1, nil, 500*ms), fetchWait(500 * ms),
			},
		},
		{
			name: "min equals max retries",
			setup: consumerSetup{
				maxAttempts: 4, minBackoff: time.Second, maxBackoff: time.Second,
				fetches:    []fetchResult{{msg: m1}},
				handleErrs: map[int64][]error{1: {errTestHandle, errTestHandle, errTestHandle}},
			},
			want: []call{
				fetched(m1, 0),
				handled(m1, errTestHandle, 0), handled(m1, errTestHandle, time.Second),
				handled(m1, errTestHandle, 2*time.Second), handled(m1, nil, 3*time.Second),
				committed(m1, nil, 3*time.Second), fetchWait(3 * time.Second),
			},
		},
		{
			name: "retry pause capped by max",
			setup: consumerSetup{
				maxAttempts: 5, minBackoff: 500 * ms, maxBackoff: time.Second,
				fetches:    []fetchResult{{msg: m1}},
				handleErrs: map[int64][]error{1: {errTestHandle, errTestHandle, errTestHandle, errTestHandle}},
			},
			// паузы 500ms, 1s, 1s, 1s
			want: []call{
				fetched(m1, 0),
				handled(m1, errTestHandle, 0), handled(m1, errTestHandle, 500*ms),
				handled(m1, errTestHandle, 1500*ms), handled(m1, errTestHandle, 2500*ms),
				handled(m1, nil, 3500*ms),
				committed(m1, nil, 3500*ms), fetchWait(3500 * ms),
			},
		},
		{
			name: "attempts exhausted go to dlq",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle, errTestHandle, errTestHandle}}
			}),
			// после третьей неудачи DLQ без паузы
			want: []call{
				fetched(m1, 0),
				handled(m1, errTestHandle, 0), handled(m1, errTestHandle, 500*ms), handled(m1, errTestHandle, 1500*ms),
				sent(m1, nil, 1500*ms), committed(m1, nil, 1500*ms), fetchWait(1500 * ms),
			},
			wantReasons: [][]error{{errTestHandle}},
		},
		{
			name: "single attempt goes to dlq immediately",
			setup: with(func(s *consumerSetup) {
				s.maxAttempts = 1
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, errTestHandle, 0),
				sent(m1, nil, 0), committed(m1, nil, 0), fetchWait(0),
			},
			wantReasons: [][]error{{errTestHandle}},
		},
		{
			name: "wrapped error exhausted keeps cause in reason",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errWrapped, errWrapped, errWrapped}}
			}),
			want: []call{
				fetched(m1, 0),
				handled(m1, errWrapped, 0), handled(m1, errWrapped, 500*ms), handled(m1, errWrapped, 1500*ms),
				sent(m1, nil, 1500*ms), committed(m1, nil, 1500*ms), fetchWait(1500 * ms),
			},
			wantReasons: [][]error{{errTestHandle, errWrapped}},
		},
		{
			name: "poison goes to dlq without retries",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {poison}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, poison, 0),
				sent(m1, nil, 0), committed(m1, nil, 0), fetchWait(0),
			},
			wantReasons: [][]error{{poison}},
		},
		{
			name: "wrapped poison goes to dlq without retries",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errPoisonWrapped}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, errPoisonWrapped, 0),
				sent(m1, nil, 0), committed(m1, nil, 0), fetchWait(0),
			},
			wantReasons: [][]error{{poison, errPoisonWrapped}},
		},
		{
			name: "poison on second message",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
				s.handleErrs = map[int64][]error{2: {poison}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, nil, 0), committed(m1, nil, 0),
				fetched(m2, 0), handled(m2, poison, 0), sent(m2, nil, 0), committed(m2, nil, 0),
				fetchWait(0),
			},
			wantReasons: [][]error{{poison}},
		},
		{
			name: "poison with single attempt",
			setup: with(func(s *consumerSetup) {
				s.maxAttempts = 1
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {poison}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, poison, 0),
				sent(m1, nil, 0), committed(m1, nil, 0), fetchWait(0),
			},
			wantReasons: [][]error{{poison}},
		},
		{
			name: "poison on second attempt stops retries",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle, poison}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, errTestHandle, 0), handled(m1, poison, 500*ms),
				sent(m1, nil, 500*ms), committed(m1, nil, 500*ms), fetchWait(500 * ms),
			},
			wantReasons: [][]error{{poison}},
		},
		{
			name: "dlq send retried until success",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {poison}}
				s.sendErrs = []error{errTestSend, errTestSend}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, poison, 0),
				sent(m1, errTestSend, 0), sent(m1, errTestSend, 500*ms), sent(m1, nil, 1500*ms),
				committed(m1, nil, 1500*ms), fetchWait(1500 * ms),
			},
			wantReasons: [][]error{{poison}, {poison}, {poison}},
		},
		{
			name: "dlq retry pause starts from min after handle retries",
			setup: with(func(s *consumerSetup) {
				s.maxAttempts = 2
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle, errTestHandle}}
				s.sendErrs = []error{errTestSend}
			}),
			// пауза DLQ 500ms, а не продолжение счётчика handle (1s)
			want: []call{
				fetched(m1, 0), handled(m1, errTestHandle, 0), handled(m1, errTestHandle, 500*ms),
				sent(m1, errTestSend, 500*ms), sent(m1, nil, 1000*ms),
				committed(m1, nil, 1000*ms), fetchWait(1000 * ms),
			},
			wantReasons: [][]error{{errTestHandle}, {errTestHandle}},
		},
		{
			name: "dlq send many errors capped by max",
			setup: with(func(s *consumerSetup) {
				s.maxBackoff = 2 * time.Second
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {poison}}
				s.sendErrs = []error{errTestSend, errTestSend, errTestSend, errTestSend, errTestSend}
			}),
			// паузы 500ms, 1s, 2s, 2s, 2s
			want: []call{
				fetched(m1, 0), handled(m1, poison, 0),
				sent(m1, errTestSend, 0), sent(m1, errTestSend, 500*ms), sent(m1, errTestSend, 1500*ms),
				sent(m1, errTestSend, 3500*ms), sent(m1, errTestSend, 5500*ms), sent(m1, nil, 7500*ms),
				committed(m1, nil, 7500*ms), fetchWait(7500 * ms),
			},
			wantReasons: [][]error{{poison}, {poison}, {poison}, {poison}, {poison}, {poison}},
		},
		{
			name: "commit retried until success then next message",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
				s.commitErrs = []error{errTestCommit, errTestCommit}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, nil, 0),
				committed(m1, errTestCommit, 0), committed(m1, errTestCommit, 500*ms), committed(m1, nil, 1500*ms),
				fetched(m2, 1500*ms), handled(m2, nil, 1500*ms), committed(m2, nil, 1500*ms),
				fetchWait(1500 * ms),
			},
		},
		{
			name: "commit after dlq retried",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {poison}}
				s.commitErrs = []error{errTestCommit}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, poison, 0), sent(m1, nil, 0),
				committed(m1, errTestCommit, 0), committed(m1, nil, 500*ms), fetchWait(500 * ms),
			},
			wantReasons: [][]error{{poison}},
		},
		{
			name: "fetch errors with growing pause",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{err: errTestFetch}, {err: errTestFetch}, {err: errTestFetch}, {msg: m1}}
			}),
			// паузы 500ms, 1s, 2s
			want: []call{
				fetchFailed(errTestFetch, 0), fetchFailed(errTestFetch, 500*ms), fetchFailed(errTestFetch, 1500*ms),
				fetched(m1, 3500*ms), handled(m1, nil, 3500*ms), committed(m1, nil, 3500*ms),
				fetchWait(3500 * ms),
			},
		},
		{
			name: "fetch errors capped by max",
			setup: with(func(s *consumerSetup) {
				s.maxBackoff = 2 * time.Second
				s.fetches = []fetchResult{
					{err: errTestFetch}, {err: errTestFetch}, {err: errTestFetch}, {err: errTestFetch}, {err: errTestFetch},
				}
			}),
			// паузы 500ms, 1s, 2s, 2s, 2s
			want: []call{
				fetchFailed(errTestFetch, 0), fetchFailed(errTestFetch, 500*ms), fetchFailed(errTestFetch, 1500*ms),
				fetchFailed(errTestFetch, 3500*ms), fetchFailed(errTestFetch, 5500*ms),
				fetchWait(7500 * ms),
			},
		},
		{
			name: "fetch pause resets after commit",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{err: errTestFetch}, {err: errTestFetch}, {msg: m1}, {err: errTestFetch}, {msg: m2}}
			}),
			want: []call{
				fetchFailed(errTestFetch, 0), fetchFailed(errTestFetch, 500*ms),
				fetched(m1, 1500*ms), handled(m1, nil, 1500*ms), committed(m1, nil, 1500*ms),
				fetchFailed(errTestFetch, 1500*ms),
				fetched(m2, 2000*ms), handled(m2, nil, 2000*ms), committed(m2, nil, 2000*ms),
				fetchWait(2000 * ms),
			},
		},
		{
			name: "fetch pause resets after dlq commit",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{err: errTestFetch}, {err: errTestFetch}, {msg: m1}, {err: errTestFetch}, {msg: m2}}
				s.handleErrs = map[int64][]error{1: {poison}}
			}),
			want: []call{
				fetchFailed(errTestFetch, 0), fetchFailed(errTestFetch, 500*ms),
				fetched(m1, 1500*ms), handled(m1, poison, 1500*ms), sent(m1, nil, 1500*ms), committed(m1, nil, 1500*ms),
				fetchFailed(errTestFetch, 1500*ms),
				fetched(m2, 2000*ms), handled(m2, nil, 2000*ms), committed(m2, nil, 2000*ms),
				fetchWait(2000 * ms),
			},
			wantReasons: [][]error{{poison}},
		},
		{
			name: "first message to dlq second succeeds with fresh pause",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
				s.handleErrs = map[int64][]error{
					1: {errTestHandle, errTestHandle, errTestHandle},
					2: {errTestHandle},
				}
			}),
			want: []call{
				fetched(m1, 0),
				handled(m1, errTestHandle, 0), handled(m1, errTestHandle, 500*ms), handled(m1, errTestHandle, 1500*ms),
				sent(m1, nil, 1500*ms), committed(m1, nil, 1500*ms),
				fetched(m2, 1500*ms), handled(m2, errTestHandle, 1500*ms), handled(m2, nil, 2000*ms),
				committed(m2, nil, 2000*ms), fetchWait(2000 * ms),
			},
			wantReasons: [][]error{{errTestHandle}},
		},
		{
			name: "dlq pause fresh for next message",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
				s.handleErrs = map[int64][]error{1: {poison}, 2: {poison}}
				s.sendErrs = []error{errTestSend, errTestSend, nil, errTestSend}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, poison, 0),
				sent(m1, errTestSend, 0), sent(m1, errTestSend, 500*ms), sent(m1, nil, 1500*ms),
				committed(m1, nil, 1500*ms),
				fetched(m2, 1500*ms), handled(m2, poison, 1500*ms),
				sent(m2, errTestSend, 1500*ms), sent(m2, nil, 2000*ms),
				committed(m2, nil, 2000*ms), fetchWait(2000 * ms),
			},
			wantReasons: [][]error{{poison}, {poison}, {poison}, {poison}, {poison}},
		},
		{
			name: "commit pause fresh for next message",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
				s.commitErrs = []error{errTestCommit, errTestCommit, nil, errTestCommit}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, nil, 0),
				committed(m1, errTestCommit, 0), committed(m1, errTestCommit, 500*ms), committed(m1, nil, 1500*ms),
				fetched(m2, 1500*ms), handled(m2, nil, 1500*ms),
				committed(m2, errTestCommit, 1500*ms), committed(m2, nil, 2000*ms),
				fetchWait(2000 * ms),
			},
		},
		{
			name: "handle pause fresh for next message",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
				s.handleErrs = map[int64][]error{1: {errTestHandle}, 2: {errTestHandle}}
			}),
			want: []call{
				fetched(m1, 0), handled(m1, errTestHandle, 0), handled(m1, nil, 500*ms), committed(m1, nil, 500*ms),
				fetched(m2, 500*ms), handled(m2, errTestHandle, 500*ms), handled(m2, nil, 1000*ms),
				committed(m2, nil, 1000*ms), fetchWait(1000 * ms),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newConsumerFixture(tt.setup)
				ctx, cancel := context.WithCancel(t.Context())
				done := runConsumer(ctx, f.consumer)

				<-f.reader.idle
				synctest.Wait()
				got := f.log.snapshot()
				cancel()
				<-done

				assert.Equal(t, tt.want, got)
				reasons := f.dlq.reasonsSnapshot()
				require.Len(t, reasons, len(tt.wantReasons))
				for i, targets := range tt.wantReasons {
					for _, target := range targets {
						assert.ErrorIsf(t, reasons[i], target, "send %d", i)
					}
				}
			})
		})
	}
}

func TestConsumer_Run_CancelDuringWait(t *testing.T) {
	const ms = time.Millisecond
	m1 := testMessage(1)
	poison := domain.ErrPoisonMessage

	with := func(mod func(s *consumerSetup)) consumerSetup {
		s := defaultSetup()
		mod(&s)
		return s
	}

	tests := []struct {
		name  string
		setup consumerSetup
		want  []call
	}{
		{
			name: "blocked fetch on empty topic",
			setup: with(func(s *consumerSetup) {
				s.stopAt = 1
			}),
			want: []call{fetchWait(0)},
		},
		{
			name: "first handle retry pause",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle, errTestHandle, errTestHandle}}
				s.stopAt = 2
			}),
			want: []call{fetched(m1, 0), handled(m1, errTestHandle, 0)},
		},
		{
			name: "second handle retry pause",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {errTestHandle, errTestHandle, errTestHandle}}
				s.stopAt = 3
			}),
			want: []call{fetched(m1, 0), handled(m1, errTestHandle, 0), handled(m1, errTestHandle, 500*ms)},
		},
		{
			name: "dlq send retry pause",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.handleErrs = map[int64][]error{1: {poison}}
				s.sendAlwaysErr = errTestSend
				s.stopAt = 3
			}),
			want: []call{fetched(m1, 0), handled(m1, poison, 0), sent(m1, errTestSend, 0)},
		},
		{
			name: "commit retry pause",
			setup: with(func(s *consumerSetup) {
				s.fetches = []fetchResult{{msg: m1}}
				s.commitAlwaysErr = errTestCommit
				s.stopAt = 3
			}),
			want: []call{fetched(m1, 0), handled(m1, nil, 0), committed(m1, errTestCommit, 0)},
		},
		{
			name: "fetch error pause",
			setup: with(func(s *consumerSetup) {
				s.fetchAlwaysErr = errTestFetch
				s.stopAt = 1
			}),
			want: []call{fetchFailed(errTestFetch, 0)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newConsumerFixture(tt.setup)
				ctx, cancel := context.WithCancel(t.Context())
				done := runConsumer(ctx, f.consumer)

				<-f.log.stop
				synctest.Wait()
				before := f.log.snapshot()
				canceledAt := f.log.elapsed()
				cancel()
				<-done

				assert.Equal(t, tt.want, before)
				// Run вернулся, не дожидаясь конца паузы
				assert.Equal(t, canceledAt, f.log.elapsed())
				// после отмены ни handle, ни DLQ, ни Commit
				assert.Equal(t, withoutFetches(before), withoutFetches(f.log.snapshot()))
			})
		})
	}
}

func TestConsumer_Run_CanceledBeforeStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newConsumerFixture(consumerSetup{
			maxAttempts: TEST_MAX_ATTEMPTS, minBackoff: TEST_MIN_BACKOFF, maxBackoff: TEST_MAX_BACKOFF,
			fetches: []fetchResult{{msg: testMessage(1)}},
		})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		done := runConsumer(ctx, f.consumer)
		<-done

		assert.Equal(t, []call(nil), withoutFetches(f.log.snapshot()))
		assert.Equal(t, time.Duration(0), f.log.elapsed())
	})
}

func TestConsumer_Run_HandleErrorAfterCancel(t *testing.T) {
	m1 := testMessage(1)
	tests := []struct {
		name      string
		handleErr error
	}{
		{name: "regular error", handleErr: errTestHandle},
		{name: "poison error", handleErr: domain.ErrPoisonMessage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newConsumerFixture(consumerSetup{
					maxAttempts: TEST_MAX_ATTEMPTS, minBackoff: TEST_MIN_BACKOFF, maxBackoff: TEST_MAX_BACKOFF,
					fetches:    []fetchResult{{msg: m1}},
					handleErrs: map[int64][]error{1: {tt.handleErr}},
				})
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				// обработчик отменяет ctx и возвращает ошибку
				f.handler.onCall = cancel

				done := runConsumer(ctx, f.consumer)
				<-done

				assert.Equal(t, []call{handled(m1, tt.handleErr, 0)}, withoutFetches(f.log.snapshot()))
				assert.Equal(t, time.Duration(0), f.log.elapsed())
			})
		})
	}
}

// checkConsumerInvariants проверяет журнал: commit только после успеха, не больше одного, msg из Fetch
func checkConsumerInvariants(t *testing.T, calls []call) {
	t.Helper()
	var current models.Message
	var prev call
	hasPrev := false
	committedCount := map[int64]int{}
	for i, c := range calls {
		if c.Op == TEST_OP_FETCH_WAIT || c.Op == TEST_OP_FETCH {
			if c.Op == TEST_OP_FETCH && c.Err == nil {
				current = c.Msg
			}
			continue
		}
		assert.Equalf(t, current, c.Msg, "call %d %s: message differs from fetched", i, c.Op)
		if c.Op == TEST_OP_COMMIT {
			prevOK := hasPrev && assert.ObjectsAreEqual(c.Msg, prev.Msg) &&
				((prev.Op == TEST_OP_HANDLE && prev.Err == nil) ||
					(prev.Op == TEST_OP_SEND && prev.Err == nil) ||
					(prev.Op == TEST_OP_COMMIT && prev.Err != nil))
			assert.Truef(t, prevOK, "call %d: commit without successful handle or dlq send", i)
			if c.Err == nil {
				committedCount[c.Msg.Offset]++
			}
		}
		prev = c
		hasPrev = true
	}
	for offset, n := range committedCount {
		assert.LessOrEqualf(t, n, 1, "message %d committed more than once", offset)
	}
}

func TestConsumer_Run_Invariants(t *testing.T) {
	m1, m2, m3 := testMessage(1), testMessage(2), testMessage(3)
	poison := domain.ErrPoisonMessage

	with := func(mod func(s *consumerSetup)) consumerSetup {
		s := defaultSetup()
		mod(&s)
		return s
	}

	tests := []struct {
		name  string
		setup consumerSetup
	}{
		{name: "three successes", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{msg: m1}, {msg: m2}, {msg: m3}}
		})},
		{name: "attempts exhausted", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{msg: m1}}
			s.handleErrs = map[int64][]error{1: {errTestHandle, errTestHandle, errTestHandle}}
		})},
		{name: "poison on second message", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
			s.handleErrs = map[int64][]error{2: {poison}}
		})},
		{name: "dlq send retried", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{msg: m1}}
			s.handleErrs = map[int64][]error{1: {poison}}
			s.sendErrs = []error{errTestSend, errTestSend}
		})},
		{name: "commit retried", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
			s.commitErrs = []error{errTestCommit, errTestCommit}
		})},
		{name: "fetch errors", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{err: errTestFetch}, {err: errTestFetch}, {err: errTestFetch}, {msg: m1}}
		})},
		{name: "dlq then success", setup: with(func(s *consumerSetup) {
			s.fetches = []fetchResult{{msg: m1}, {msg: m2}}
			s.handleErrs = map[int64][]error{
				1: {errTestHandle, errTestHandle, errTestHandle},
				2: {errTestHandle},
			}
		})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newConsumerFixture(tt.setup)
				ctx, cancel := context.WithCancel(t.Context())
				done := runConsumer(ctx, f.consumer)

				<-f.reader.idle
				synctest.Wait()
				got := f.log.snapshot()
				cancel()
				<-done

				checkConsumerInvariants(t, got)
			})
		})
	}
}
