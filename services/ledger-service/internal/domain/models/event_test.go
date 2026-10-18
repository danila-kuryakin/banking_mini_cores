package models

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_TRANSACTION_ID   = "0190f1a2-0000-7000-8000-000000000001"
	TEST_FROM_ACCOUNT_ID  = "0190f1a2-0000-7000-8000-0000000000a1"
	TEST_TO_ACCOUNT_ID    = "0190f1a2-0000-7000-8000-0000000000b2"
	TEST_CASH_ACCOUNT_ID  = "0190f1a2-0000-7000-8000-0000000000c3"
	TEST_INITIATOR_ID     = "0190f1a2-0000-7000-8000-0000000000d4"
	TEST_OFFICER_ID       = "0190f1a2-0000-7000-8000-0000000000e5"
	TEST_FROM_USER_ID     = "0190f1a2-0000-7000-8000-0000000000f6"
	TEST_TO_USER_ID       = "0190f1a2-0000-7000-8000-000000000107"
	TEST_NIL_UUID         = "00000000-0000-0000-0000-000000000000"
	TEST_DECLINE_REASON   = "insufficient funds"
	TEST_AMOUNT           = int64(1500)
	TEST_CURRENCY_RUB     = Currency("RUB")
	TEST_ENTRY_ID_DEBIT   = "0190f1a2-0000-7000-8000-000000000201"
	TEST_ENTRY_ID_CREDIT  = "0190f1a2-0000-7000-8000-000000000202"
	TEST_FIXED_ZONE_NAME  = "MSK"
	TEST_FIXED_ZONE_SHIFT = 3 * 60 * 60
)

// TEST_CREATED_AT — фиксированное время создания транзакции
var TEST_CREATED_AT = time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)

// uuidPtr возвращает указатель на разобранный UUID
func uuidPtr(raw string) *uuid.UUID {
	parsed := uuid.MustParse(raw)
	return &parsed
}

// testEntries — проводки, которые не должны попасть в payload
func testEntries() []Entry {
	return []Entry{
		{ID: uuid.MustParse(TEST_ENTRY_ID_DEBIT), TransactionID: uuid.MustParse(TEST_TRANSACTION_ID), AccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), Direction: DIRECTION_DEBIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, CreatedAt: TEST_CREATED_AT},
		{ID: uuid.MustParse(TEST_ENTRY_ID_CREDIT), TransactionID: uuid.MustParse(TEST_TRANSACTION_ID), AccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID), Direction: DIRECTION_CREDIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, CreatedAt: TEST_CREATED_AT},
	}
}

// baseKeys — ключи payload, которые есть всегда
var baseKeys = []string{"amount", "created_at", "currency", "from_account_id", "status", "to_account_id", "transaction_id", "type"}

// withKeys дополняет базовые ключи и сортирует результат
func withKeys(extra ...string) []string {
	keys := append(append([]string{}, baseKeys...), extra...)
	sort.Strings(keys)
	return keys
}

func TestNewTransactionEvent(t *testing.T) {
	tests := []struct {
		name          string
		tx            Transaction
		wantKey       string
		wantEventType string
		wantPayload   TransactionEvent
		wantKeys      []string
	}{
		{
			name: "transfer completed with all fields",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_TRANSFER, Status: STATUS_COMPLETED,
				FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, InitiatorID: uuidPtr(TEST_INITIATOR_ID),
				CreatedAt: TEST_CREATED_AT, Entries: testEntries(),
				FromUserID: uuidPtr(TEST_FROM_USER_ID), ToUserID: uuidPtr(TEST_TO_USER_ID),
			},
			wantKey:       TEST_FROM_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_COMPLETED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "transfer", Status: "completed",
				FromAccountID: TEST_FROM_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", InitiatorID: TEST_INITIATOR_ID,
				FromUserID: TEST_FROM_USER_ID, ToUserID: TEST_TO_USER_ID, CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys("initiator_id", "from_user_id", "to_user_id"),
		},
		{
			name: "deposit completed without from user",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_DEPOSIT, Status: STATUS_COMPLETED,
				FromAccountID: uuid.MustParse(TEST_CASH_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, InitiatorID: uuidPtr(TEST_OFFICER_ID),
				CreatedAt: TEST_CREATED_AT, ToUserID: uuidPtr(TEST_TO_USER_ID),
			},
			wantKey:       TEST_TO_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_COMPLETED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "deposit", Status: "completed",
				FromAccountID: TEST_CASH_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", InitiatorID: TEST_OFFICER_ID,
				ToUserID: TEST_TO_USER_ID, CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys("initiator_id", "to_user_id"),
		},
		{
			name: "transfer declined with reason",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_TRANSFER, Status: STATUS_DECLINED,
				FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, InitiatorID: uuidPtr(TEST_INITIATOR_ID),
				DeclineReason: TEST_DECLINE_REASON, CreatedAt: TEST_CREATED_AT,
				FromUserID: uuidPtr(TEST_FROM_USER_ID), ToUserID: uuidPtr(TEST_TO_USER_ID),
			},
			wantKey:       TEST_FROM_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_DECLINED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "transfer", Status: "declined",
				FromAccountID: TEST_FROM_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", InitiatorID: TEST_INITIATOR_ID,
				FromUserID: TEST_FROM_USER_ID, ToUserID: TEST_TO_USER_ID,
				DeclineReason: TEST_DECLINE_REASON, CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys("initiator_id", "from_user_id", "to_user_id", "decline_reason"),
		},
		{
			name: "deposit declined keyed by receiver",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_DEPOSIT, Status: STATUS_DECLINED,
				FromAccountID: uuid.MustParse(TEST_CASH_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, InitiatorID: uuidPtr(TEST_OFFICER_ID),
				DeclineReason: TEST_DECLINE_REASON, CreatedAt: TEST_CREATED_AT, ToUserID: uuidPtr(TEST_TO_USER_ID),
			},
			wantKey:       TEST_TO_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_DECLINED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "deposit", Status: "declined",
				FromAccountID: TEST_CASH_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", InitiatorID: TEST_OFFICER_ID,
				ToUserID: TEST_TO_USER_ID, DeclineReason: TEST_DECLINE_REASON, CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys("initiator_id", "to_user_id", "decline_reason"),
		},
		{
			name: "nil pointers and empty reason omitted",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_TRANSFER, Status: STATUS_COMPLETED,
				FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, CreatedAt: TEST_CREATED_AT, Entries: testEntries(),
			},
			wantKey:       TEST_FROM_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_COMPLETED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "transfer", Status: "completed",
				FromAccountID: TEST_FROM_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys(),
		},
		{
			name: "zero amount kept in payload",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_TRANSFER, Status: STATUS_DECLINED,
				FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: 0, Currency: TEST_CURRENCY_RUB, CreatedAt: TEST_CREATED_AT,
			},
			wantKey:       TEST_FROM_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_DECLINED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "transfer", Status: "declined",
				FromAccountID: TEST_FROM_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: 0, Currency: "RUB", CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys(),
		},
		{
			name: "nil from account rendered as zero uuid",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_DEPOSIT, Status: STATUS_COMPLETED,
				FromAccountID: uuid.Nil, ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, CreatedAt: TEST_CREATED_AT,
			},
			wantKey:       TEST_TO_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_COMPLETED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "deposit", Status: "completed",
				FromAccountID: TEST_NIL_UUID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys(),
		},
		{
			name: "created at in non utc zone",
			tx: Transaction{
				ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_TRANSFER, Status: STATUS_COMPLETED,
				FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
				Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB,
				CreatedAt: TEST_CREATED_AT.In(time.FixedZone(TEST_FIXED_ZONE_NAME, TEST_FIXED_ZONE_SHIFT)),
			},
			wantKey:       TEST_FROM_ACCOUNT_ID,
			wantEventType: EVENT_TRANSACTION_COMPLETED,
			wantPayload: TransactionEvent{
				TransactionID: TEST_TRANSACTION_ID, Type: "transfer", Status: "completed",
				FromAccountID: TEST_FROM_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID,
				Amount: TEST_AMOUNT, Currency: "RUB", CreatedAt: TEST_CREATED_AT,
			},
			wantKeys: withKeys(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := tt.tx
			msg, err := NewTransactionEvent(&tx)
			require.NoError(t, err)

			assert.Equal(t, tt.wantKey, msg.Key)
			assert.Equal(t, tt.wantEventType, msg.EventType)
			assert.True(t, tt.tx.CreatedAt.Equal(msg.OccurredAt), "occurred_at %v must equal created_at %v", msg.OccurredAt, tt.tx.CreatedAt)

			var got TransactionEvent
			require.NoError(t, json.Unmarshal(msg.Payload, &got))
			assert.True(t, tt.wantPayload.CreatedAt.Equal(got.CreatedAt), "payload created_at %v must equal %v", got.CreatedAt, tt.wantPayload.CreatedAt)
			// время сравнено через Equal, остальное сравниваем структурой целиком
			want := tt.wantPayload
			want.CreatedAt = time.Time{}
			got.CreatedAt = time.Time{}
			assert.Equal(t, want, got)

			var raw map[string]any
			require.NoError(t, json.Unmarshal(msg.Payload, &raw))
			gotKeys := make([]string, 0, len(raw))
			for key := range raw {
				gotKeys = append(gotKeys, key)
			}
			sort.Strings(gotKeys)
			assert.Equal(t, tt.wantKeys, gotKeys)
			assert.Equal(t, float64(tt.wantPayload.Amount), raw["amount"])
		})
	}
}

func TestNewTransactionEvent_UniqueV7IDs(t *testing.T) {
	tx := Transaction{
		ID: uuid.MustParse(TEST_TRANSACTION_ID), Type: TYPE_TRANSFER, Status: STATUS_COMPLETED,
		FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID), ToAccountID: uuid.MustParse(TEST_TO_ACCOUNT_ID),
		Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, CreatedAt: TEST_CREATED_AT,
	}

	first, err := NewTransactionEvent(&tx)
	require.NoError(t, err)
	second, err := NewTransactionEvent(&tx)
	require.NoError(t, err)

	assert.Equal(t, uuid.Version(7), first.ID.Version())
	assert.Equal(t, uuid.Version(7), second.ID.Version())
	assert.NotEqual(t, first.ID, second.ID)
}
