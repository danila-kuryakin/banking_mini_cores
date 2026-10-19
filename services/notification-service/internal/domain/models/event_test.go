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
	TEST_EVENT_ID   = "11111111-1111-4111-8111-111111111111"
	TEST_EVENT_TYPE = "ledger.transaction_completed"
	TEST_NIL_UUID   = "00000000-0000-0000-0000-000000000000"
)

// Тест разбора конверта: успешные кейсы и ошибки без синтаксических
func TestParseEnvelope(t *testing.T) {
	eventID := uuid.MustParse(TEST_EVENT_ID)
	occurredAt := time.Date(2026, 9, 29, 1, 30, 1, 0, time.UTC)

	tests := []struct {
		name    string
		raw     []byte
		want    Envelope
		wantErr bool
	}{
		{
			name: "full envelope",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed","occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100}}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, OccurredAt: occurredAt, Payload: json.RawMessage(`{"amount":100}`)},
		},
		{
			// 04:30:01+03:00 = 01:30:01Z
			name: "occurred_at with offset",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed","occurred_at":"2026-09-29T04:30:01+03:00","payload":{"amount":100}}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, OccurredAt: occurredAt, Payload: json.RawMessage(`{"amount":100}`)},
		},
		{
			name: "payload bytes kept as is",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed","occurred_at":"2026-09-29T01:30:01Z","payload":{ "amount" : 100 }}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, OccurredAt: occurredAt, Payload: json.RawMessage(`{ "amount" : 100 }`)},
		},
		{
			name: "payload is array",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed","occurred_at":"2026-09-29T01:30:01Z","payload":[1,2]}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, OccurredAt: occurredAt, Payload: json.RawMessage(`[1,2]`)},
		},
		{
			name: "extra fields ignored",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","extra":1,"event_type":"ledger.transaction_completed","occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100},"source":"ledger"}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, OccurredAt: occurredAt, Payload: json.RawMessage(`{"amount":100}`)},
		},
		{
			name: "no occurred_at and no payload",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed"}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE},
		},
		{
			name: "no occurred_at",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed","payload":{"amount":100}}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, Payload: json.RawMessage(`{"amount":100}`)},
		},
		{
			name: "no payload",
			raw:  []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"ledger.transaction_completed","occurred_at":"2026-09-29T01:30:01Z"}`),
			want: Envelope{EventID: eventID, EventType: TEST_EVENT_TYPE, OccurredAt: occurredAt},
		},
		{name: "event_id not uuid", raw: []byte(`{"event_id":"not-a-uuid","event_type":"x"}`), wantErr: true},
		{name: "event_id is number", raw: []byte(`{"event_id":123,"event_type":"x"}`), wantErr: true},
		{name: "event_type is number", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":5}`), wantErr: true},
		{name: "occurred_at not rfc3339", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"x","occurred_at":"29.09.2026"}`), wantErr: true},
		{name: "root is array", raw: []byte(`[]`), wantErr: true},
		{name: "event_id missing", raw: []byte(`{"event_type":"x"}`), wantErr: true},
		{name: "event_id nil uuid", raw: []byte(`{"event_id":"` + TEST_NIL_UUID + `","event_type":"x"}`), wantErr: true},
		{name: "event_type missing", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111"}`), wantErr: true},
		{name: "event_type empty", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":""}`), wantErr: true},
		{name: "empty object", raw: []byte(`{}`), wantErr: true},
		{name: "json null", raw: []byte(`null`), wantErr: true},
		{name: "empty slice", raw: []byte{}, wantErr: true},
		{name: "nil input", raw: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvelope(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, Envelope{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.EventID, got.EventID)
			assert.Equal(t, tt.want.EventType, got.EventType)
			assert.True(t, tt.want.OccurredAt.Equal(got.OccurredAt), "occurred_at: want %v, got %v", tt.want.OccurredAt, got.OccurredAt)
			assert.Equal(t, string(tt.want.Payload), string(got.Payload))
		})
	}
}

// Синтаксическая ошибка JSON оборачивается и видна через errors.As
func TestParseEnvelope_SyntaxError(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "open brace", raw: []byte(`{`)},
		{name: "garbage", raw: []byte(`not json`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvelope(tt.raw)
			var syntaxErr *json.SyntaxError
			require.ErrorAs(t, err, &syntaxErr)
			assert.Equal(t, Envelope{}, got)
		})
	}
}

// Инвариант: при любой ошибке результат равен Envelope{}
func TestParseEnvelope_ErrorReturnsZeroEnvelope(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "open brace", raw: []byte(`{`)},
		{name: "garbage", raw: []byte(`not json`)},
		{name: "event_id not uuid", raw: []byte(`{"event_id":"not-a-uuid","event_type":"x","occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100}}`)},
		{name: "event_type wrong type", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":5,"occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100}}`)},
		{name: "occurred_at not rfc3339", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"x","occurred_at":"29.09.2026","payload":{"amount":100}}`)},
		{name: "nil uuid with other fields", raw: []byte(`{"event_id":"` + TEST_NIL_UUID + `","event_type":"x","occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100}}`)},
		{name: "empty type with other fields", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","event_type":"","occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100}}`)},
		{name: "missing type with other fields", raw: []byte(`{"event_id":"11111111-1111-4111-8111-111111111111","occurred_at":"2026-09-29T01:30:01Z","payload":{"amount":100}}`)},
		{name: "root is array", raw: []byte(`[]`)},
		{name: "empty object", raw: []byte(`{}`)},
		{name: "json null", raw: []byte(`null`)},
		{name: "empty slice", raw: []byte{}},
		{name: "nil input", raw: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvelope(tt.raw)
			require.Error(t, err)
			assert.Equal(t, Envelope{}, got)
		})
	}
}
