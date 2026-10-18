package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

const TEST_RELAY_BATCH_SIZE = 2

var TEST_ERR_KAFKA = errors.New("kafka write failed")

// testOutboxMessage - событие outbox с заданным ключом
func testOutboxMessage(eventID uuid.UUID, key string) models.OutboxMessage {
	return models.OutboxMessage{
		ID:         eventID,
		Key:        key,
		EventType:  "kyc.application_submitted",
		Payload:    json.RawMessage(`{"status":"submitted"}`),
		OccurredAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
	}
}

// newTestOutboxRelay - воркер поверх фейков
func newTestOutboxRelay(outbox *fakeOutboxRepository, publisher *fakeEventPublisher) *OutboxRelay {
	return NewOutboxRelay(outbox, publisher, slog.New(slog.DiscardHandler), TEST_INTERVAL, TEST_RELAY_BATCH_SIZE)
}

func TestOutboxRelay_drain(t *testing.T) {
	msgs1 := []models.OutboxMessage{
		testOutboxMessage(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"), TEST_USER_ID.String()),
		testOutboxMessage(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"), TEST_USER_ID_2.String()),
	}
	msgs2 := []models.OutboxMessage{
		testOutboxMessage(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3"), TEST_USER_ID_3.String()),
	}

	tests := []struct {
		name          string
		responses     []relayResponse
		publishErr    error
		wantRelays    []int
		wantPublishes [][]models.OutboxMessage
	}{
		{
			name:       "full, full, partial - three batches",
			responses:  []relayResponse{{sent: 2}, {sent: 2}, {sent: 1}},
			wantRelays: []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE},
		},
		{
			name:       "full then empty - two batches",
			responses:  []relayResponse{{sent: 2}, {sent: 0}},
			wantRelays: []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE},
		},
		{
			name:       "empty - one batch",
			responses:  []relayResponse{{sent: 0}},
			wantRelays: []int{TEST_RELAY_BATCH_SIZE},
		},
		{
			name:       "relay error - stops after first",
			responses:  []relayResponse{{err: TEST_ERR_DB}},
			wantRelays: []int{TEST_RELAY_BATCH_SIZE},
		},
		{
			name:       "full then relay error - stops after second",
			responses:  []relayResponse{{sent: 2}, {err: TEST_ERR_DB}},
			wantRelays: []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE},
		},
		{
			name:          "messages reach publisher unchanged",
			responses:     []relayResponse{{sent: 2, msgs: msgs1}, {sent: 1, msgs: msgs2}},
			wantRelays:    []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE},
			wantPublishes: [][]models.OutboxMessage{msgs1, msgs2},
		},
		{
			name:          "publish error - stops after first",
			responses:     []relayResponse{{sent: 2, msgs: msgs1}, {sent: 1, msgs: msgs2}},
			publishErr:    TEST_ERR_KAFKA,
			wantRelays:    []int{TEST_RELAY_BATCH_SIZE},
			wantPublishes: [][]models.OutboxMessage{msgs1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outbox := &fakeOutboxRepository{responses: tt.responses}
			publisher := &fakeEventPublisher{publishErr: tt.publishErr}
			w := newTestOutboxRelay(outbox, publisher)

			w.drain(t.Context())

			assert.Equal(t, tt.wantRelays, outbox.relays())
			assert.Equal(t, tt.wantPublishes, publisher.publishes())
		})
	}
}

// startOutboxRelay - запускает Run в горутине; done закрывается при возврате
func startOutboxRelay(ctx context.Context, w *OutboxRelay) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()
	return done
}

func TestOutboxRelay_Run_TicksAndStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		outbox := &fakeOutboxRepository{}
		w := newTestOutboxRelay(outbox, &fakeEventPublisher{})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := startOutboxRelay(ctx, w)

		time.Sleep(TEST_INTERVAL - time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, []int(nil), outbox.relays())

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, []int{TEST_RELAY_BATCH_SIZE}, outbox.relays())

		time.Sleep(2 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE}, outbox.relays())
		requireRunning(t, done)

		cancel()
		synctest.Wait()
		requireDone(t, done)

		time.Sleep(2 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE}, outbox.relays())
	})
}

func TestOutboxRelay_Run_DrainsWholeOnTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// первый тик: полная пачка и неполная - два вызова; второй тик: пусто - один
		outbox := &fakeOutboxRepository{responses: []relayResponse{{sent: 2}, {sent: 1}, {sent: 0}}}
		w := newTestOutboxRelay(outbox, &fakeEventPublisher{})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := startOutboxRelay(ctx, w)

		time.Sleep(TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE}, outbox.relays())

		time.Sleep(TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE}, outbox.relays())

		cancel()
		synctest.Wait()
		requireDone(t, done)
	})
}

func TestOutboxRelay_Run_ContinuesAfterError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		outbox := &fakeOutboxRepository{responses: []relayResponse{{err: TEST_ERR_DB}, {err: TEST_ERR_DB}, {err: TEST_ERR_DB}}}
		w := newTestOutboxRelay(outbox, &fakeEventPublisher{})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := startOutboxRelay(ctx, w)

		time.Sleep(3 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE, TEST_RELAY_BATCH_SIZE}, outbox.relays())
		requireRunning(t, done)

		cancel()
		synctest.Wait()
		requireDone(t, done)
	})
}
