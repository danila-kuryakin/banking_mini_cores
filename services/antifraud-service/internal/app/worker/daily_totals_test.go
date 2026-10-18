package worker

import (
	"encoding/json"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

const (
	TEST_TOPIC          = "ledger.events"
	TEST_PARTITION      = 3
	TEST_EVENT_ID_1     = "11111111-1111-4111-8111-111111111111"
	TEST_EVENT_ID_2     = "22222222-2222-4222-8222-222222222222"
	TEST_EVENT_ID_3     = "33333333-3333-4333-8333-333333333333"
	TEST_EVENT_ID_4     = "44444444-4444-4444-8444-444444444444"
	TEST_ACCOUNT_ID     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	TEST_TO_ACCOUNT_ID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	TEST_TRANSACTION_ID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	TEST_NIL_UUID       = "00000000-0000-0000-0000-000000000000"
	TEST_AMOUNT         = int64(1500)
	TEST_CREATED_AT     = "2026-09-28T23:30:00-02:00"
	TEST_OCCURRED_AT    = "2026-09-29T01:30:01Z"
	TEST_MIN_BACKOFF    = 500 * time.Millisecond
	TEST_MAX_BACKOFF    = 4 * time.Second
	TEST_TYPE_DEPOSIT   = "deposit"
	TEST_TYPE_DECLINED  = "ledger.transaction_declined"
	TEST_TYPE_KYC       = "kyc.application_approved"
)

// newMessage - сообщение топика с заданным оффсетом и телом
func newMessage(offset int64, value string) models.Message {
	return models.Message{
		Topic:     TEST_TOPIC,
		Partition: TEST_PARTITION,
		Offset:    offset,
		Key:       []byte(TEST_ACCOUNT_ID),
		Value:     []byte(value),
	}
}

// transferPayload - валидный payload перевода с заменой и удалением ключей
func transferPayload(overrides map[string]any, removed ...string) map[string]any {
	payload := map[string]any{
		"transaction_id":  TEST_TRANSACTION_ID,
		"type":            models.TRANSACTION_TYPE_TRANSFER,
		"from_account_id": TEST_ACCOUNT_ID,
		"to_account_id":   TEST_TO_ACCOUNT_ID,
		"amount":          TEST_AMOUNT,
		"currency":        "rub",
		"created_at":      TEST_CREATED_AT,
	}
	for key, value := range overrides {
		payload[key] = value
	}
	for _, key := range removed {
		delete(payload, key)
	}
	return payload
}

// eventValue - JSON-конверт события ledger.events
func eventValue(t *testing.T, eventID, eventType string, payload any) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"event_id":    eventID,
		"event_type":  eventType,
		"occurred_at": TEST_OCCURRED_AT,
		"payload":     payload,
	})
	require.NoError(t, err)
	return string(body)
}

// validValue - валидное событие перевода
func validValue(t *testing.T, eventID string) string {
	t.Helper()
	return eventValue(t, eventID, models.EVENT_TRANSACTION_COMPLETED, transferPayload(nil))
}

// wantTransfer - ожидаемый аргумент AddTransfer для валидного события
func wantTransfer(eventID string, currency models.Currency, day time.Time) models.CompletedTransfer {
	return models.CompletedTransfer{
		EventID:   uuid.MustParse(eventID),
		EventType: models.EVENT_TRANSACTION_COMPLETED,
		AccountID: uuid.MustParse(TEST_ACCOUNT_ID),
		Currency:  currency,
		Amount:    TEST_AMOUNT,
		Day:       day,
	}
}

// repeatTransfer - n одинаковых вызовов AddTransfer
func repeatTransfer(transfer models.CompletedTransfer, n int) []models.CompletedTransfer {
	out := make([]models.CompletedTransfer, 0, n)
	for range n {
		out = append(out, transfer)
	}
	return out
}

func TestDailyTotals_Run_Processing(t *testing.T) {
	// 23:30 -02:00 28-го = 01:30Z 29-го
	day29 := time.Date(2026, time.September, 29, 0, 0, 0, 0, time.UTC)
	day28 := time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC)

	valid1 := newMessage(1, validValue(t, TEST_EVENT_ID_1))
	valid2 := newMessage(2, validValue(t, TEST_EVENT_ID_2))
	valid3 := newMessage(3, validValue(t, TEST_EVENT_ID_3))
	midnight := newMessage(1, eventValue(t, TEST_EVENT_ID_1, models.EVENT_TRANSACTION_COMPLETED,
		transferPayload(map[string]any{"created_at": "2026-09-29T00:00:00Z"})))
	lastSecond := newMessage(1, eventValue(t, TEST_EVENT_ID_1, models.EVENT_TRANSACTION_COMPLETED,
		transferPayload(map[string]any{"created_at": "2026-09-28T23:59:59Z"})))
	paddedCurrency := newMessage(1, eventValue(t, TEST_EVENT_ID_1, models.EVENT_TRANSACTION_COMPLETED,
		transferPayload(map[string]any{"currency": " usd "})))
	extraKeys := newMessage(1, `{"event_id":"`+TEST_EVENT_ID_1+`","event_type":"ledger.transaction_completed",`+
		`"occurred_at":"`+TEST_OCCURRED_AT+`","extra":{"a":1},"payload":{"transaction_id":"`+TEST_TRANSACTION_ID+`",`+
		`"type":"transfer","from_account_id":"`+TEST_ACCOUNT_ID+`","to_account_id":"`+TEST_TO_ACCOUNT_ID+`",`+
		`"amount":1500,"currency":"rub","created_at":"`+TEST_CREATED_AT+`","from_user_id":"x","note":[1,2]}}`)
	declined := newMessage(1, eventValue(t, TEST_EVENT_ID_1, TEST_TYPE_DECLINED, transferPayload(nil)))
	kyc := newMessage(1, eventValue(t, TEST_EVENT_ID_1, TEST_TYPE_KYC, map[string]any{"status": "approved"}))
	deposit := newMessage(1, eventValue(t, TEST_EVENT_ID_1, models.EVENT_TRANSACTION_COMPLETED,
		transferPayload(map[string]any{"type": TEST_TYPE_DEPOSIT})))
	// поток из примера п.6: poison, valid1, deposit, valid2
	streamPoison := newMessage(1, "{not json")
	streamValid1 := newMessage(2, validValue(t, TEST_EVENT_ID_1))
	streamDeposit := newMessage(3, eventValue(t, TEST_EVENT_ID_3, models.EVENT_TRANSACTION_COMPLETED,
		transferPayload(map[string]any{"type": TEST_TYPE_DEPOSIT})))
	streamValid2 := newMessage(4, validValue(t, TEST_EVENT_ID_2))

	tests := []struct {
		name                string
		fetches             []fetchResult
		addResults          []addResult
		wantAdds            []models.CompletedTransfer
		wantCommits         []models.Message
		wantCommitAddCounts []int
		wantFetchCount      int
	}{
		{
			name:                "valid transfer is added and committed",
			fetches:             []fetchResult{{msg: valid1}},
			wantAdds:            []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_1, "RUB", day29)},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      2,
		},
		{
			name:                "created_at at utc midnight stays on same day",
			fetches:             []fetchResult{{msg: midnight}},
			wantAdds:            []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_1, "RUB", day29)},
			wantCommits:         []models.Message{midnight},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      2,
		},
		{
			name:                "created_at at last second of day",
			fetches:             []fetchResult{{msg: lastSecond}},
			wantAdds:            []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_1, "RUB", day28)},
			wantCommits:         []models.Message{lastSecond},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      2,
		},
		{
			name:                "currency is trimmed and uppercased",
			fetches:             []fetchResult{{msg: paddedCurrency}},
			wantAdds:            []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_1, "USD", day29)},
			wantCommits:         []models.Message{paddedCurrency},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      2,
		},
		{
			name:                "unknown keys are ignored",
			fetches:             []fetchResult{{msg: extraKeys}},
			wantAdds:            []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_1, "RUB", day29)},
			wantCommits:         []models.Message{extraKeys},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      2,
		},
		{
			name:    "messages are processed and committed in fetch order",
			fetches: []fetchResult{{msg: valid1}, {msg: valid2}, {msg: valid3}},
			wantAdds: []models.CompletedTransfer{
				wantTransfer(TEST_EVENT_ID_1, "RUB", day29),
				wantTransfer(TEST_EVENT_ID_2, "RUB", day29),
				wantTransfer(TEST_EVENT_ID_3, "RUB", day29),
			},
			wantCommits:         []models.Message{valid1, valid2, valid3},
			wantCommitAddCounts: []int{1, 2, 3},
			wantFetchCount:      4,
		},
		{
			name:                "duplicate event is committed without retry",
			fetches:             []fetchResult{{msg: valid1}},
			addResults:          []addResult{{added: false}},
			wantAdds:            []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_1, "RUB", day29)},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      2,
		},
		{
			name:                "declined transaction is skipped",
			fetches:             []fetchResult{{msg: declined}},
			wantCommits:         []models.Message{declined},
			wantCommitAddCounts: []int{0},
			wantFetchCount:      2,
		},
		{
			name:                "foreign event type is skipped",
			fetches:             []fetchResult{{msg: kyc}},
			wantCommits:         []models.Message{kyc},
			wantCommitAddCounts: []int{0},
			wantFetchCount:      2,
		},
		{
			name:                "deposit is skipped",
			fetches:             []fetchResult{{msg: deposit}},
			wantCommits:         []models.Message{deposit},
			wantCommitAddCounts: []int{0},
			wantFetchCount:      2,
		},
		{
			name: "mixed stream from example",
			fetches: []fetchResult{
				{msg: streamPoison}, {msg: streamValid1}, {msg: streamDeposit}, {msg: streamValid2},
			},
			wantAdds: []models.CompletedTransfer{
				wantTransfer(TEST_EVENT_ID_1, "RUB", day29),
				wantTransfer(TEST_EVENT_ID_2, "RUB", day29),
			},
			wantCommits:         []models.Message{streamPoison, streamValid1, streamDeposit, streamValid2},
			wantCommitAddCounts: []int{0, 1, 1, 2},
			wantFetchCount:      5,
		},
		{
			name:           "cancel while waiting in fetch",
			wantFetchCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				totals := &fakeTotals{results: tt.addResults}
				reader := &fakeReader{fetchQueue: tt.fetches, totals: totals}

				runWorker(t, reader, totals, TEST_MIN_BACKOFF, TEST_MAX_BACKOFF, 0)

				assert.Equal(t, tt.wantAdds, totals.Calls())
				assert.Equal(t, tt.wantCommits, reader.Commits())
				assert.Equal(t, tt.wantCommitAddCounts, reader.CommitAddCounts())
				assert.Len(t, reader.FetchTimes(), tt.wantFetchCount)
			})
		})
	}
}

func TestDailyTotals_Run_Poison(t *testing.T) {
	// 23:30 -02:00 28-го = 01:30Z 29-го
	day29 := time.Date(2026, time.September, 29, 0, 0, 0, 0, time.UTC)
	completed := models.EVENT_TRANSACTION_COMPLETED
	withPayload := func(overrides map[string]any, removed ...string) string {
		return eventValue(t, TEST_EVENT_ID_1, completed, transferPayload(overrides, removed...))
	}
	noEventID, err := json.Marshal(map[string]any{
		"event_type": completed, "occurred_at": TEST_OCCURRED_AT, "payload": transferPayload(nil),
	})
	require.NoError(t, err)
	noEventType, err := json.Marshal(map[string]any{
		"event_id": TEST_EVENT_ID_1, "occurred_at": TEST_OCCURRED_AT, "payload": transferPayload(nil),
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid json", value: "{not json"},
		{name: "empty value", value: ""},
		{name: "null value", value: "null"},
		{name: "array instead of object", value: "[1,2]"},
		{name: "missing event_id", value: string(noEventID)},
		{name: "nil event_id", value: eventValue(t, TEST_NIL_UUID, completed, transferPayload(nil))},
		{name: "event_id is not uuid", value: eventValue(t, "not-a-uuid", completed, transferPayload(nil))},
		{name: "empty event_type", value: eventValue(t, TEST_EVENT_ID_1, "", transferPayload(nil))},
		{name: "missing event_type", value: string(noEventType)},
		{name: "payload is string", value: eventValue(t, TEST_EVENT_ID_1, completed, "oops")},
		{name: "amount is string", value: withPayload(map[string]any{"amount": "1500"})},
		{name: "amount is fractional", value: withPayload(map[string]any{"amount": 15.5})},
		{name: "currency of two letters", value: withPayload(map[string]any{"currency": "RU"})},
		{name: "currency with digit", value: withPayload(map[string]any{"currency": "R1B"})},
		{name: "empty currency", value: withPayload(map[string]any{"currency": ""})},
		{name: "currency of four letters", value: withPayload(map[string]any{"currency": "RUBL"})},
		{name: "currency in cyrillic", value: withPayload(map[string]any{"currency": "РУБ"})},
		{name: "from_account_id is not uuid", value: withPayload(map[string]any{"from_account_id": "not-a-uuid"})},
		{name: "created_at is not rfc3339", value: withPayload(map[string]any{"created_at": "yesterday"})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				poison := newMessage(1, tt.value)
				valid := newMessage(2, validValue(t, TEST_EVENT_ID_2))
				totals := &fakeTotals{}
				reader := &fakeReader{fetchQueue: []fetchResult{{msg: poison}, {msg: valid}}, totals: totals}

				runWorker(t, reader, totals, TEST_MIN_BACKOFF, TEST_MAX_BACKOFF, 0)

				// poison не доходит до репозитория, следующее сообщение обрабатывается
				assert.Equal(t, []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_2, "RUB", day29)}, totals.Calls())
				assert.Equal(t, []models.Message{poison, valid}, reader.Commits())
				assert.Equal(t, []int{0, 1}, reader.CommitAddCounts())
				assert.Len(t, reader.FetchTimes(), 3)
			})
		})
	}
}

func TestDailyTotals_Run_Retries(t *testing.T) {
	// 23:30 -02:00 28-го = 01:30Z 29-го
	day29 := time.Date(2026, time.September, 29, 0, 0, 0, 0, time.UTC)
	transfer1 := wantTransfer(TEST_EVENT_ID_1, "RUB", day29)
	transfer2 := wantTransfer(TEST_EVENT_ID_2, "RUB", day29)
	valid1 := newMessage(1, validValue(t, TEST_EVENT_ID_1))
	valid2 := newMessage(2, validValue(t, TEST_EVENT_ID_2))
	poison := newMessage(1, "{not json")

	tests := []struct {
		name                string
		minBackoff          time.Duration
		maxBackoff          time.Duration
		fetches             []fetchResult
		addResults          []addResult
		addErrAlways        error
		commitErrs          []error
		commitErrAlways     error
		advance             time.Duration
		wantAdds            []models.CompletedTransfer
		wantAddPauses       []time.Duration
		wantCommits         []models.Message
		wantCommitPauses    []time.Duration
		wantCommitAddCounts []int
		wantFetchCount      int
		wantFetchPauses     []time.Duration
	}{
		{
			name:       "add errors back off from min doubling up to max",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 4 * time.Second,
			fetches:    []fetchResult{{msg: valid1}},
			addResults: []addResult{{err: errDB}, {err: errDB}, {err: errDB}, {err: errDB}, {err: errDB}},
			advance:    20 * time.Second,
			wantAdds:   repeatTransfer(transfer1, 6),
			wantAddPauses: []time.Duration{
				500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second,
			},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{6},
			wantFetchCount:      2,
			// 0.5+1+2+4+4 = 11.5s до коммита и следующего Fetch
			wantFetchPauses: []time.Duration{11500 * time.Millisecond},
		},
		{
			name:                "min equals max",
			minBackoff:          time.Second,
			maxBackoff:          time.Second,
			fetches:             []fetchResult{{msg: valid1}},
			addResults:          []addResult{{err: errDB}, {err: errDB}, {err: errDB}},
			advance:             10 * time.Second,
			wantAdds:            repeatTransfer(transfer1, 4),
			wantAddPauses:       []time.Duration{time.Second, time.Second, time.Second},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{4},
			wantFetchCount:      2,
			wantFetchPauses:     []time.Duration{3 * time.Second},
		},
		{
			name:       "max not multiple of min",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 3 * time.Second,
			fetches:    []fetchResult{{msg: valid1}},
			addResults: []addResult{{err: errDB}, {err: errDB}, {err: errDB}, {err: errDB}, {err: errDB}},
			advance:    20 * time.Second,
			wantAdds:   repeatTransfer(transfer1, 6),
			wantAddPauses: []time.Duration{
				500 * time.Millisecond, time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second,
			},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{6},
			wantFetchCount:      2,
			// 0.5+1+2+3+3 = 9.5s
			wantFetchPauses: []time.Duration{9500 * time.Millisecond},
		},
		{
			name:       "cancel during add backoff returns without commit",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 4 * time.Second,
			fetches:    []fetchResult{{msg: valid1}},
			// вызовы в 0 и 500ms, отмена на 700ms посреди паузы 1s
			addErrAlways:   errDB,
			advance:        700 * time.Millisecond,
			wantAdds:       repeatTransfer(transfer1, 2),
			wantAddPauses:  []time.Duration{500 * time.Millisecond},
			wantFetchCount: 1,
		},
		{
			name:       "fetch errors back off and fetch again",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 4 * time.Second,
			fetches: []fetchResult{
				{err: errDB}, {err: errDB}, {err: errDB}, {msg: valid1},
			},
			advance:             10 * time.Second,
			wantAdds:            []models.CompletedTransfer{transfer1},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      5,
			wantFetchPauses:     []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second, 0},
		},
		{
			name:       "fetch backoff restarts from min after commit",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 4 * time.Second,
			fetches: []fetchResult{
				{err: errDB}, {err: errDB}, {msg: valid1}, {err: errDB},
			},
			advance:             10 * time.Second,
			wantAdds:            []models.CompletedTransfer{transfer1},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      5,
			wantFetchPauses:     []time.Duration{500 * time.Millisecond, time.Second, 0, 500 * time.Millisecond},
		},
		{
			name:       "add backoff starts from min after fetch errors",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 4 * time.Second,
			fetches: []fetchResult{
				{err: errDB}, {err: errDB}, {msg: valid1},
			},
			addResults:          []addResult{{err: errDB}},
			advance:             10 * time.Second,
			wantAdds:            repeatTransfer(transfer1, 2),
			wantAddPauses:       []time.Duration{500 * time.Millisecond},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{2},
			wantFetchCount:      4,
			// Fetch в 0, 0.5s, 1.5s, затем после паузы AddTransfer в 2s
			wantFetchPauses: []time.Duration{500 * time.Millisecond, time.Second, 500 * time.Millisecond},
		},
		{
			name:           "cancel during fetch backoff",
			minBackoff:     500 * time.Millisecond,
			maxBackoff:     4 * time.Second,
			fetches:        []fetchResult{{err: errDB}},
			advance:        200 * time.Millisecond,
			wantFetchCount: 1,
		},
		{
			name:                "commit error retries same message without new add",
			minBackoff:          500 * time.Millisecond,
			maxBackoff:          4 * time.Second,
			fetches:             []fetchResult{{msg: valid1}},
			commitErrs:          []error{errDB, errDB},
			advance:             10 * time.Second,
			wantAdds:            []models.CompletedTransfer{transfer1},
			wantCommits:         []models.Message{valid1, valid1, valid1},
			wantCommitPauses:    []time.Duration{500 * time.Millisecond, time.Second},
			wantCommitAddCounts: []int{1, 1, 1},
			wantFetchCount:      2,
			wantFetchPauses:     []time.Duration{1500 * time.Millisecond},
		},
		{
			name:                "cancel during commit backoff",
			minBackoff:          500 * time.Millisecond,
			maxBackoff:          4 * time.Second,
			fetches:             []fetchResult{{msg: valid1}},
			commitErrAlways:     errDB,
			advance:             200 * time.Millisecond,
			wantAdds:            []models.CompletedTransfer{transfer1},
			wantCommits:         []models.Message{valid1},
			wantCommitAddCounts: []int{1},
			wantFetchCount:      1,
		},
		{
			name:                "commit error of poison message is retried",
			minBackoff:          500 * time.Millisecond,
			maxBackoff:          4 * time.Second,
			fetches:             []fetchResult{{msg: poison}},
			commitErrs:          []error{errDB},
			advance:             10 * time.Second,
			wantCommits:         []models.Message{poison, poison},
			wantCommitPauses:    []time.Duration{500 * time.Millisecond},
			wantCommitAddCounts: []int{0, 0},
			wantFetchCount:      2,
			wantFetchPauses:     []time.Duration{500 * time.Millisecond},
		},
		{
			name:       "add backoff restarts from min for next message",
			minBackoff: 500 * time.Millisecond,
			maxBackoff: 4 * time.Second,
			fetches:    []fetchResult{{msg: valid1}, {msg: valid2}},
			addResults: []addResult{{err: errDB}, {err: errDB}, {added: true}, {err: errDB}},
			advance:    10 * time.Second,
			wantAdds:   []models.CompletedTransfer{transfer1, transfer1, transfer1, transfer2, transfer2},
			// valid1 в 0, 0.5s, 1.5s; valid2 в 1.5s и 2s
			wantAddPauses:       []time.Duration{500 * time.Millisecond, time.Second, 0, 500 * time.Millisecond},
			wantCommits:         []models.Message{valid1, valid2},
			wantCommitPauses:    []time.Duration{500 * time.Millisecond},
			wantCommitAddCounts: []int{3, 5},
			wantFetchCount:      3,
			wantFetchPauses:     []time.Duration{1500 * time.Millisecond, 500 * time.Millisecond},
		},
		{
			name:                "commit backoff starts from min after add retries",
			minBackoff:          500 * time.Millisecond,
			maxBackoff:          4 * time.Second,
			fetches:             []fetchResult{{msg: valid1}},
			addResults:          []addResult{{err: errDB}},
			commitErrs:          []error{errDB},
			advance:             10 * time.Second,
			wantAdds:            repeatTransfer(transfer1, 2),
			wantAddPauses:       []time.Duration{500 * time.Millisecond},
			wantCommits:         []models.Message{valid1, valid1},
			wantCommitPauses:    []time.Duration{500 * time.Millisecond},
			wantCommitAddCounts: []int{2, 2},
			wantFetchCount:      2,
			wantFetchPauses:     []time.Duration{time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				totals := &fakeTotals{results: tt.addResults, errAlways: tt.addErrAlways}
				reader := &fakeReader{
					fetchQueue:      tt.fetches,
					commitErrs:      tt.commitErrs,
					commitErrAlways: tt.commitErrAlways,
					totals:          totals,
				}

				runWorker(t, reader, totals, tt.minBackoff, tt.maxBackoff, tt.advance)

				assert.Equal(t, tt.wantAdds, totals.Calls())
				assert.Equal(t, tt.wantAddPauses, pauses(totals.CallTimes()))
				assert.Equal(t, tt.wantCommits, reader.Commits())
				assert.Equal(t, tt.wantCommitPauses, pauses(reader.CommitTimes()))
				assert.Equal(t, tt.wantCommitAddCounts, reader.CommitAddCounts())
				assert.Len(t, reader.FetchTimes(), tt.wantFetchCount)
				assert.Equal(t, tt.wantFetchPauses, pauses(reader.FetchTimes()))
			})
		})
	}
}

func TestDailyTotals_Run_Invariants(t *testing.T) {
	valid1 := newMessage(1, validValue(t, TEST_EVENT_ID_1))
	valid2 := newMessage(2, validValue(t, TEST_EVENT_ID_2))
	valid3 := newMessage(3, validValue(t, TEST_EVENT_ID_3))
	valid4 := newMessage(4, validValue(t, TEST_EVENT_ID_4))
	duplicate := newMessage(5, validValue(t, TEST_EVENT_ID_1))
	poison1 := newMessage(1, "{not json")
	poison2 := newMessage(2, "null")
	poison3 := newMessage(3, eventValue(t, TEST_EVENT_ID_3, models.EVENT_TRANSACTION_COMPLETED, "oops"))
	declined := newMessage(2, eventValue(t, TEST_EVENT_ID_2, TEST_TYPE_DECLINED, transferPayload(nil)))
	deposit := newMessage(3, eventValue(t, TEST_EVENT_ID_3, models.EVENT_TRANSACTION_COMPLETED,
		transferPayload(map[string]any{"type": TEST_TYPE_DEPOSIT})))
	poison4 := newMessage(4, "")
	mixedValid := newMessage(5, validValue(t, TEST_EVENT_ID_4))

	tests := []struct {
		name         string
		messages     []models.Message
		addResults   []addResult
		wantAddCount int
	}{
		{name: "empty stream", messages: nil, wantAddCount: 0},
		{name: "only poison", messages: []models.Message{poison1, poison2, poison3}, wantAddCount: 0},
		{name: "mixed", messages: []models.Message{valid1, declined, deposit, poison4, mixedValid}, wantAddCount: 2},
		{name: "only valid", messages: []models.Message{valid1, valid2, valid3, valid4}, wantAddCount: 4},
		{
			name:         "with duplicates",
			messages:     []models.Message{valid1, duplicate},
			addResults:   []addResult{{added: true}, {added: false}},
			wantAddCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				fetches := make([]fetchResult, 0, len(tt.messages))
				for _, msg := range tt.messages {
					fetches = append(fetches, fetchResult{msg: msg})
				}
				totals := &fakeTotals{results: tt.addResults}
				reader := &fakeReader{fetchQueue: fetches, totals: totals}

				runWorker(t, reader, totals, TEST_MIN_BACKOFF, TEST_MAX_BACKOFF, 0)

				// коммитов столько же, сколько сообщений, и в том же порядке
				assert.Equal(t, tt.messages, reader.Commits())
				assert.Len(t, totals.Calls(), tt.wantAddCount)
			})
		})
	}
}

// R13: подозрительный перевод - poison, в дневную сумму не попадает
func TestDailyTotals_Run_SuspiciousTransferIsPoison(t *testing.T) {
	// 23:30 -02:00 28-го = 01:30Z 29-го
	day29 := time.Date(2026, time.September, 29, 0, 0, 0, 0, time.UTC)
	completed := models.EVENT_TRANSACTION_COMPLETED

	tests := []struct {
		name  string
		value string
	}{
		{name: "missing from_account_id", value: eventValue(t, TEST_EVENT_ID_1, completed, transferPayload(nil, "from_account_id"))},
		{name: "nil from_account_id", value: eventValue(t, TEST_EVENT_ID_1, completed, transferPayload(map[string]any{"from_account_id": TEST_NIL_UUID}))},
		{name: "zero amount", value: eventValue(t, TEST_EVENT_ID_1, completed, transferPayload(map[string]any{"amount": 0}))},
		{name: "negative amount", value: eventValue(t, TEST_EVENT_ID_1, completed, transferPayload(map[string]any{"amount": -100}))},
		{name: "missing created_at", value: eventValue(t, TEST_EVENT_ID_1, completed, transferPayload(nil, "created_at"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				suspicious := newMessage(1, tt.value)
				valid := newMessage(2, validValue(t, TEST_EVENT_ID_2))
				totals := &fakeTotals{}
				reader := &fakeReader{fetchQueue: []fetchResult{{msg: suspicious}, {msg: valid}}, totals: totals}

				runWorker(t, reader, totals, TEST_MIN_BACKOFF, TEST_MAX_BACKOFF, 0)

				assert.Equal(t, []models.CompletedTransfer{wantTransfer(TEST_EVENT_ID_2, "RUB", day29)}, totals.Calls())
				assert.Equal(t, []models.Message{suspicious, valid}, reader.Commits())
				assert.Equal(t, []int{0, 1}, reader.CommitAddCounts())
			})
		})
	}
}
