package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

const (
	TEST_BATCH_SIZE    = 3
	TEST_INTERVAL      = time.Second
	TEST_ERROR_LOG     = `level=ERROR msg="failed to relay outbox"`
	TEST_ERROR_LEVEL   = "level=ERROR"
	TEST_EVENT_TYPE    = "ledger.transaction_completed"
	TEST_MESSAGE_KEY_1 = "8d1f6b0e-4a51-4c1e-9b8e-0a6f2c7d1e01"
	TEST_MESSAGE_KEY_2 = "8d1f6b0e-4a51-4c1e-9b8e-0a6f2c7d1e02"
)

var errRelay = errors.New("relay failed")

// newTestLogger создаёт логгер уровня Debug поверх потокобезопасного буфера
func newTestLogger() (*slog.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

// testMessages возвращает две полностью заполненные записи outbox
func testMessages() []models.OutboxMessage {
	occurredAt := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	return []models.OutboxMessage{
		{
			ID:         uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			Key:        TEST_MESSAGE_KEY_1,
			EventType:  TEST_EVENT_TYPE,
			Payload:    json.RawMessage(`{"amount":100}`),
			OccurredAt: occurredAt,
		},
		{
			ID:         uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Key:        TEST_MESSAGE_KEY_2,
			EventType:  TEST_EVENT_TYPE,
			Payload:    json.RawMessage(`{"amount":200}`),
			OccurredAt: occurredAt.Add(time.Minute),
		},
	}
}

func TestOutboxRelay_drain_PassesPublisherAndLimit(t *testing.T) {
	msgs := testMessages()
	repo := &fakeOutboxRepository{
		responses:   []relayResponse{{sent: 2}},
		publishMsgs: msgs,
	}
	publisher := &fakeEventPublisher{}
	log, buf := newTestLogger()
	w := NewOutboxRelay(repo, publisher, log, TEST_INTERVAL, TEST_BATCH_SIZE)

	w.drain(t.Context())

	assert.Equal(t, []int{TEST_BATCH_SIZE}, repo.gotLimits())
	assert.Equal(t, [][]models.OutboxMessage{testMessages()}, publisher.gotBatches())
	assert.Equal(t, []error{nil}, repo.gotPublishErrs())
	assert.NotContains(t, buf.String(), TEST_ERROR_LEVEL)
}

func TestOutboxRelay_drain_PublisherErrorReachesRelay(t *testing.T) {
	errPublish := errors.New("kafka down")
	repo := &fakeOutboxRepository{
		publishMsgs:         testMessages(),
		propagatePublishErr: true,
	}
	publisher := &fakeEventPublisher{publishErr: errPublish}
	log, buf := newTestLogger()
	w := NewOutboxRelay(repo, publisher, log, TEST_INTERVAL, TEST_BATCH_SIZE)

	w.drain(t.Context())

	publishErrs := repo.gotPublishErrs()
	require.Len(t, publishErrs, 1)
	assert.ErrorIs(t, publishErrs[0], errPublish)
	assert.Equal(t, 1, repo.calls())
	assert.Contains(t, buf.String(), TEST_ERROR_LOG)
}

func TestOutboxRelay_drain_BatchSizeOne(t *testing.T) {
	repo := &fakeOutboxRepository{responses: []relayResponse{{sent: 1}, {sent: 0}}}
	log, _ := newTestLogger()
	w := NewOutboxRelay(repo, &fakeEventPublisher{}, log, TEST_INTERVAL, 1)

	w.drain(t.Context())

	assert.Equal(t, []int{1, 1}, repo.gotLimits())
}

func TestOutboxRelay_drain(t *testing.T) {
	tests := []struct {
		name         string
		responses    []relayResponse
		wantCalls    int
		wantErrorLog bool
	}{
		{name: "empty batch", responses: []relayResponse{{sent: 0}}, wantCalls: 1},
		{name: "partial batch", responses: []relayResponse{{sent: 2}}, wantCalls: 1},
		{name: "full then partial", responses: []relayResponse{{sent: 3}, {sent: 1}}, wantCalls: 2},
		{name: "two full then empty", responses: []relayResponse{{sent: 3}, {sent: 3}, {sent: 0}}, wantCalls: 3},
		// после двух полных пачек очередь пуста, фейк отдаёт (0, nil)
		{name: "two full then exhausted queue", responses: []relayResponse{{sent: 3}, {sent: 3}}, wantCalls: 3},
		{name: "full then error", responses: []relayResponse{{sent: 3}, {err: errRelay}}, wantCalls: 2, wantErrorLog: true},
		{name: "error first", responses: []relayResponse{{err: errRelay}}, wantCalls: 1, wantErrorLog: true},
		{name: "error with full batch", responses: []relayResponse{{sent: 3, err: errRelay}}, wantCalls: 1, wantErrorLog: true},
		{name: "context canceled", responses: []relayResponse{{err: context.Canceled}}, wantCalls: 1},
		{name: "wrapped context canceled", responses: []relayResponse{{err: fmt.Errorf("relay: %w", context.Canceled)}}, wantCalls: 1},
		{name: "full then context canceled", responses: []relayResponse{{sent: 3}, {err: context.Canceled}}, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeOutboxRepository{responses: tt.responses}
			log, buf := newTestLogger()
			w := NewOutboxRelay(repo, &fakeEventPublisher{}, log, TEST_INTERVAL, TEST_BATCH_SIZE)

			w.drain(t.Context())

			wantLimits := make([]int, tt.wantCalls)
			for i := range wantLimits {
				wantLimits[i] = TEST_BATCH_SIZE
			}
			assert.Equal(t, wantLimits, repo.gotLimits())
			logText := buf.String()
			assert.Equal(t, tt.wantErrorLog, strings.Contains(logText, TEST_ERROR_LOG))
			assert.Equal(t, tt.wantErrorLog, strings.Contains(logText, TEST_ERROR_LEVEL))
		})
	}
}

// runStep - сдвиг виртуального времени и ожидаемое число вызовов Relay после него
type runStep struct {
	advance   time.Duration
	wantCalls int
}

func TestOutboxRelay_Run(t *testing.T) {
	tests := []struct {
		name         string
		responses    []relayResponse
		steps        []runStep
		wantErrorLog bool
	}{
		{
			name:      "one drain per interval",
			responses: []relayResponse{{sent: 0}},
			// время нарастает: 999ms, 1s, 3s
			steps: []runStep{{advance: 999 * time.Millisecond, wantCalls: 0}, {advance: time.Millisecond, wantCalls: 1}, {advance: 2 * time.Second, wantCalls: 3}},
		},
		{
			name:      "full batch drained within one tick",
			responses: []relayResponse{{sent: 3}, {sent: 1}},
			steps:     []runStep{{advance: time.Second, wantCalls: 2}, {advance: time.Second, wantCalls: 3}},
		},
		{
			name:         "error does not stop worker",
			responses:    []relayResponse{{err: errRelay}, {sent: 0}},
			steps:        []runStep{{advance: time.Second, wantCalls: 1}, {advance: time.Second, wantCalls: 2}},
			wantErrorLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				repo := &fakeOutboxRepository{responses: tt.responses}
				log, buf := newTestLogger()
				w := NewOutboxRelay(repo, &fakeEventPublisher{}, log, TEST_INTERVAL, TEST_BATCH_SIZE)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				var returned atomic.Bool
				go func() {
					w.Run(ctx)
					returned.Store(true)
				}()

				for _, step := range tt.steps {
					time.Sleep(step.advance)
					synctest.Wait()
					assert.Equal(t, step.wantCalls, repo.calls(), "after advance %s", step.advance)
				}
				callsBeforeCancel := repo.calls()

				cancel()
				synctest.Wait()
				require.True(t, returned.Load(), "Run did not return after cancel")

				time.Sleep(5 * TEST_INTERVAL)
				synctest.Wait()
				assert.Equal(t, callsBeforeCancel, repo.calls())
				assert.Equal(t, tt.wantErrorLog, strings.Contains(buf.String(), TEST_ERROR_LOG))
			})
		})
	}
}

func TestOutboxRelay_Run_CanceledBeforeStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := &fakeOutboxRepository{}
		log, _ := newTestLogger()
		w := NewOutboxRelay(repo, &fakeEventPublisher{}, log, TEST_INTERVAL, TEST_BATCH_SIZE)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		var returned atomic.Bool
		go func() {
			w.Run(ctx)
			returned.Store(true)
		}()
		synctest.Wait()
		require.True(t, returned.Load(), "Run did not return with canceled context")

		time.Sleep(5 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, 0, repo.calls())
	})
}
