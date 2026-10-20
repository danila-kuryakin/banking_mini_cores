package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_EVENT_ID        = "6f1c2b8e-3a4d-4e5f-8a9b-0c1d2e3f4a5b"
	TEST_TRANSACTION_ID  = "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
	TEST_FROM_ACCOUNT_ID = "11111111-2222-4333-8444-555555555555"
)

func TestEventConstants(t *testing.T) {
	assert.Equal(t, "ledger.transaction_completed", EVENT_TRANSACTION_COMPLETED)
	assert.Equal(t, "transfer", TRANSACTION_TYPE_TRANSFER)
}

func TestParseEnvelope(t *testing.T) {
	eventID := uuid.MustParse(TEST_EVENT_ID)
	// 15:30 в зоне +05:00 это 10:30 UTC
	plus5 := time.FixedZone("+05:00", 5*3600)

	tests := []struct {
		name        string
		raw         string
		wantID      uuid.UUID
		wantType    string
		wantAt      time.Time
		wantPayload string
		wantErr     bool
	}{
		{
			name:        "full envelope",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"2026-09-28T15:30:00Z","payload":{"amount":100}}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
			wantPayload: `{"amount":100}`,
		},
		{
			name:        "payload kept byte for byte",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"2026-09-28T15:30:00Z","payload":{ "amount" : 100 }}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
			wantPayload: `{ "amount" : 100 }`,
		},
		{
			name:        "occurred_at missing",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","payload":{"amount":100}}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Time{},
			wantPayload: `{"amount":100}`,
		},
		{
			name:        "payload null",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"2026-09-28T15:30:00Z","payload":null}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
			wantPayload: `null`,
		},
		{
			name:        "payload missing",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"2026-09-28T15:30:00Z"}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
			wantPayload: ``,
		},
		{
			name:        "payload empty object",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"2026-09-28T15:30:00Z","payload":{}}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
			wantPayload: `{}`,
		},
		{
			name:        "occurred_at with zone",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"2026-09-28T15:30:00+05:00","payload":{}}`,
			wantID:      eventID,
			wantType:    EVENT_TRANSACTION_COMPLETED,
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, plus5),
			wantPayload: `{}`,
		},
		{
			name:        "unknown event type accepted",
			raw:         `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.something_new","occurred_at":"2026-09-28T15:30:00Z","payload":{}}`,
			wantID:      eventID,
			wantType:    "ledger.something_new",
			wantAt:      time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
			wantPayload: `{}`,
		},
		{name: "invalid json", raw: `{"event_id":`, wantErr: true},
		{name: "empty input", raw: ``, wantErr: true},
		{name: "json array", raw: `[]`, wantErr: true},
		{name: "json string", raw: `"x"`, wantErr: true},
		{name: "event_id missing", raw: `{"event_type":"ledger.transaction_completed","payload":{}}`, wantErr: true},
		{name: "event_id nil uuid", raw: `{"event_id":"00000000-0000-0000-0000-000000000000","event_type":"ledger.transaction_completed","payload":{}}`, wantErr: true},
		{name: "event_id not uuid", raw: `{"event_id":"not-a-uuid","event_type":"ledger.transaction_completed","payload":{}}`, wantErr: true},
		{name: "event_id empty", raw: `{"event_id":"","event_type":"ledger.transaction_completed","payload":{}}`, wantErr: true},
		{name: "event_type missing", raw: `{"event_id":"` + TEST_EVENT_ID + `","payload":{}}`, wantErr: true},
		{name: "event_type empty", raw: `{"event_id":"` + TEST_EVENT_ID + `","event_type":"","payload":{}}`, wantErr: true},
		{name: "occurred_at not rfc3339", raw: `{"event_id":"` + TEST_EVENT_ID + `","event_type":"ledger.transaction_completed","occurred_at":"yesterday","payload":{}}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvelope([]byte(tt.raw))
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, Envelope{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, got.EventID)
			assert.Equal(t, tt.wantType, got.EventType)
			assert.True(t, tt.wantAt.Equal(got.OccurredAt), "occurred_at: want %v, got %v", tt.wantAt, got.OccurredAt)
			assert.Equal(t, tt.wantPayload, string(got.Payload))
		})
	}
}

func TestTransactionPayload_Unmarshal(t *testing.T) {
	want := TransactionPayload{
		TransactionID: TEST_TRANSACTION_ID,
		Type:          TRANSACTION_TYPE_TRANSFER,
		FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID),
		Amount:        15000,
		Currency:      "RUB",
		CreatedAt:     time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC),
	}
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "contract keys only",
			raw:  `{"transaction_id":"` + TEST_TRANSACTION_ID + `","type":"transfer","from_account_id":"` + TEST_FROM_ACCOUNT_ID + `","amount":15000,"currency":"RUB","created_at":"2026-09-28T15:30:00Z"}`,
		},
		{
			name: "extra keys ignored",
			raw:  `{"transaction_id":"` + TEST_TRANSACTION_ID + `","type":"transfer","from_account_id":"` + TEST_FROM_ACCOUNT_ID + `","to_account_id":"22222222-3333-4444-8555-666666666666","from_user_id":"33333333-4444-4555-8666-777777777777","to_user_id":"44444444-5555-4666-8777-888888888888","status":"completed","amount":15000,"currency":"RUB","created_at":"2026-09-28T15:30:00Z"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TransactionPayload
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &got))
			assert.Equal(t, want, got)
		})
	}
}

func TestDay(t *testing.T) {
	plus5 := time.FixedZone("+05:00", 5*3600)
	minus3 := time.FixedZone("-03:00", -3*3600)
	minus1 := time.FixedZone("-01:00", -1*3600)

	tests := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{name: "middle of day", at: time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC), want: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)},
		{name: "midnight", at: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC), want: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)},
		{name: "last nanosecond", at: time.Date(2026, 9, 28, 23, 59, 59, 999999999, time.UTC), want: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)},
		// 02:30 +05:00 это 21:30 UTC 27-го
		{name: "plus5 previous utc day", at: time.Date(2026, 9, 28, 2, 30, 0, 0, plus5), want: time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)},
		// 22:00 -03:00 это 01:00 UTC 28-го
		{name: "minus3 next utc day", at: time.Date(2026, 9, 27, 22, 0, 0, 0, minus3), want: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)},
		{name: "leap day", at: time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC), want: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)},
		// 23:30 -01:00 31 декабря это 00:30 UTC 1 января
		{name: "year boundary via zone", at: time.Date(2026, 12, 31, 23, 30, 0, 0, minus1), want: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "zero time", at: time.Time{}, want: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Day(tt.at)
			assert.Equal(t, time.UTC, got.Location())
			assert.True(t, tt.want.Equal(got), "want %v, got %v", tt.want, got)
		})
	}
}

func TestDay_Invariants(t *testing.T) {
	plus5 := time.FixedZone("+05:00", 5*3600)
	minus3 := time.FixedZone("-03:00", -3*3600)

	tests := []struct {
		name string
		at   time.Time
	}{
		{name: "midnight", at: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)},
		{name: "last nanosecond", at: time.Date(2026, 9, 28, 23, 59, 59, 999999999, time.UTC)},
		{name: "plus5", at: time.Date(2026, 9, 28, 2, 30, 0, 0, plus5)},
		{name: "minus3", at: time.Date(2026, 9, 27, 22, 0, 0, 0, minus3)},
		{name: "leap day", at: time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			day := Day(tt.at)
			assert.True(t, Day(day).Equal(day), "day is not idempotent: %v vs %v", Day(day), day)
			assert.False(t, day.After(tt.at), "day %v is after %v", day, tt.at)
			assert.True(t, tt.at.Before(day.Add(24*time.Hour)), "%v is not before %v", tt.at, day.Add(24*time.Hour))
		})
	}
}
