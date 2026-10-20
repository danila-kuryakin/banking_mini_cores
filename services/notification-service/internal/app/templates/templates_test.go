package templates

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

const (
	TEST_EVENT_ID        = "11111111-1111-1111-1111-111111111111"
	TEST_USER_1          = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	TEST_USER_2          = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"
	TEST_USER_3          = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3"
	TEST_BAD_UUID        = "not-a-uuid"
	TEST_FROM_ACCOUNT    = "A1"
	TEST_TO_ACCOUNT      = "A2"
	TEST_TRANSACTION_ID  = "tx-1"
	TEST_APPLICATION_ID  = "app-1"
	TEST_BROKEN_JSON     = `{broken`
	TEST_DECLINE_REASON  = "insufficient funds"
	TEST_UNKNOWN_EVENT   = "ledger.transaction_reversed"
	TEST_UNKNOWN_KYC     = "kyc.something_new"
	TEST_INVARIANT_EVENT = "22222222-2222-2222-2222-222222222222"
)

var (
	testEventID = uuid.MustParse(TEST_EVENT_ID)
	testUser1   = uuid.MustParse(TEST_USER_1)
	testUser2   = uuid.MustParse(TEST_USER_2)
)

// envelope собирает конверт с фиксированными EventID и OccurredAt
func envelope(eventID uuid.UUID, eventType string, payload json.RawMessage) models.Envelope {
	return models.Envelope{
		EventID:    eventID,
		EventType:  eventType,
		OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Payload:    payload,
	}
}

// mustJSON сериализует payload для конверта
func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}

// note собирает ожидаемое уведомление с нулевыми ID и CreatedAt
func note(userID uuid.UUID, eventType, text string) models.Notification {
	return models.Notification{
		UserID:    userID,
		EventID:   testEventID,
		EventType: eventType,
		Text:      text,
	}
}

// nonNil приводит nil-срез к пустому: R9/R12 не различают nil и пустой результат
func nonNil(notifications []models.Notification) []models.Notification {
	return append([]models.Notification{}, notifications...)
}

// transfer возвращает перевод A1 -> A2 на 150050 RUB
func transfer(fromUserID, initiatorID, toUserID string) models.TransactionPayload {
	return models.TransactionPayload{
		TransactionID: TEST_TRANSACTION_ID,
		Type:          models.TRANSACTION_TYPE_TRANSFER,
		FromAccountID: TEST_FROM_ACCOUNT,
		ToAccountID:   TEST_TO_ACCOUNT,
		Amount:        150050,
		Currency:      "RUB",
		InitiatorID:   initiatorID,
		FromUserID:    fromUserID,
		ToUserID:      toUserID,
	}
}

// declined возвращает отклонённый перевод A1 -> A2 на 100 USD
func declined(fromUserID, initiatorID, toUserID string) models.TransactionPayload {
	return models.TransactionPayload{
		TransactionID: TEST_TRANSACTION_ID,
		Type:          models.TRANSACTION_TYPE_TRANSFER,
		FromAccountID: TEST_FROM_ACCOUNT,
		ToAccountID:   TEST_TO_ACCOUNT,
		Amount:        100,
		Currency:      "USD",
		InitiatorID:   initiatorID,
		FromUserID:    fromUserID,
		ToUserID:      toUserID,
		DeclineReason: TEST_DECLINE_REASON,
	}
}

// deposit возвращает пополнение счёта A2
func deposit(toUserID string, amount int64, currency string) models.TransactionPayload {
	return models.TransactionPayload{
		TransactionID: TEST_TRANSACTION_ID,
		Type:          models.TRANSACTION_TYPE_DEPOSIT,
		ToAccountID:   TEST_TO_ACCOUNT,
		Amount:        amount,
		Currency:      currency,
		InitiatorID:   TEST_USER_3,
		ToUserID:      toUserID,
	}
}

// application возвращает payload события заявки
func application(userID, status, reason string, requestedTypes []string) models.ApplicationPayload {
	return models.ApplicationPayload{
		ApplicationID:  TEST_APPLICATION_ID,
		UserID:         userID,
		Status:         status,
		Reason:         reason,
		RequestedTypes: requestedTypes,
	}
}

const (
	TEST_TRANSFER_OUT = "Перевод 1500.50 RUB на счёт A2 выполнен."
	TEST_TRANSFER_IN  = "Поступление 1500.50 RUB на счёт A2 со счёта A1."
	TEST_DECLINED     = "Перевод 1.00 USD на счёт A2 отклонён: insufficient funds."
)

func TestBuild_Ledger(t *testing.T) {
	completed := models.EVENT_TRANSACTION_COMPLETED
	declinedType := models.EVENT_TRANSACTION_DECLINED

	selfTransfer := transfer(TEST_USER_1, "", TEST_USER_1)
	selfTransfer.Amount = 1999
	selfTransfer.Currency = "EUR"
	selfTransferByInitiator := transfer("", TEST_USER_1, TEST_USER_1)
	selfTransferByInitiator.Amount = 1999
	selfTransferByInitiator.Currency = "EUR"
	selfText := "Перевод 19.99 EUR между своими счетами выполнен: A1 -> A2."

	typeEmpty := transfer(TEST_USER_1, "", TEST_USER_2)
	typeEmpty.Type = ""
	typeRefund := transfer(TEST_USER_1, "", TEST_USER_2)
	typeRefund.Type = "refund"
	typeUpper := transfer(TEST_USER_1, "", TEST_USER_2)
	typeUpper.Type = "Transfer"

	tests := []struct {
		name      string
		eventType string
		payload   func(t *testing.T) json.RawMessage
		want      []models.Notification
		wantErr   error
	}{
		{
			name:      "L1 deposit notifies recipient only",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, deposit(TEST_USER_2, 150050, "RUB")) },
			want:      []models.Notification{note(testUser2, completed, "Счёт A2 пополнен на 1500.50 RUB.")},
		},
		{
			name:      "L2 deposit without recipient",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, deposit("", 150050, "RUB")) },
			want:      nil,
		},
		{
			name:      "L3 deposit with invalid recipient uuid",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, deposit(TEST_BAD_UUID, 150050, "RUB")) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L4 transfer notifies sender then recipient",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_USER_1, "", TEST_USER_2)) },
			want: []models.Notification{
				note(testUser1, completed, TEST_TRANSFER_OUT),
				note(testUser2, completed, TEST_TRANSFER_IN),
			},
		},
		{
			name:      "L5 transfer sender falls back to initiator",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer("", TEST_USER_1, TEST_USER_2)) },
			want: []models.Notification{
				note(testUser1, completed, TEST_TRANSFER_OUT),
				note(testUser2, completed, TEST_TRANSFER_IN),
			},
		},
		{
			name:      "L6 transfer without sender and initiator",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer("", "", TEST_USER_2)) },
			want:      []models.Notification{note(testUser2, completed, TEST_TRANSFER_IN)},
		},
		{
			name:      "L7 transfer without recipient",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_USER_1, "", "")) },
			want:      []models.Notification{note(testUser1, completed, TEST_TRANSFER_OUT)},
		},
		{
			name:      "L8 transfer without any users",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer("", "", "")) },
			want:      nil,
		},
		{
			name:      "L9 transfer between own accounts",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, selfTransfer) },
			want:      []models.Notification{note(testUser1, completed, selfText)},
		},
		{
			name:      "L10 own accounts transfer with sender from initiator",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, selfTransferByInitiator) },
			want:      []models.Notification{note(testUser1, completed, selfText)},
		},
		{
			name:      "L11 from_user_id takes precedence over initiator",
			eventType: completed,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, transfer(TEST_USER_1, TEST_USER_3, TEST_USER_2))
			},
			want: []models.Notification{
				note(testUser1, completed, TEST_TRANSFER_OUT),
				note(testUser2, completed, TEST_TRANSFER_IN),
			},
		},
		{
			name:      "L12 completed with empty type",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, typeEmpty) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L13 completed with unknown type",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, typeRefund) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L14 completed with type in other case",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, typeUpper) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L15 transfer with invalid recipient uuid",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_USER_1, "", TEST_BAD_UUID)) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L16 transfer with invalid sender uuid",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_BAD_UUID, "", TEST_USER_2)) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L17 transfer with invalid initiator uuid as sender",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer("", TEST_BAD_UUID, TEST_USER_2)) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L18 completed with wrong amount type",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(`{"amount":"x"}`) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L19 completed with broken json",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L19a completed with empty payload",
			eventType: completed,
			payload:   func(t *testing.T) json.RawMessage { return nil },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L20 declined notifies sender only",
			eventType: declinedType,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, declined(TEST_USER_1, "", TEST_USER_2)) },
			want:      []models.Notification{note(testUser1, declinedType, TEST_DECLINED)},
		},
		{
			name:      "L21 declined sender falls back to initiator",
			eventType: declinedType,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, declined("", TEST_USER_1, TEST_USER_2)) },
			want:      []models.Notification{note(testUser1, declinedType, TEST_DECLINED)},
		},
		{
			name:      "L22 declined without sender",
			eventType: declinedType,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, declined("", "", TEST_USER_2)) },
			want:      nil,
		},
		{
			name:      "L23 declined with broken json",
			eventType: declinedType,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "L24 declined with invalid sender uuid",
			eventType: declinedType,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, declined(TEST_BAD_UUID, "", TEST_USER_2)) },
			wantErr:   domain.ErrPoisonMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(envelope(testEventID, tt.eventType, tt.payload(t)))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, nonNil(tt.want), nonNil(got))
		})
	}
}

func TestBuild_Amount(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		want     string
	}{
		{name: "S1 rubles with kopecks", amount: 150050, currency: "RUB", want: "1500.50 RUB"},
		{name: "S2 exact one unit", amount: 100, currency: "USD", want: "1.00 USD"},
		{name: "S3 single digit minor", amount: 5, currency: "RUB", want: "0.05 RUB"},
		// 10/100=0, 10%100=10
		{name: "S4 two digit minor", amount: 10, currency: "RUB", want: "0.10 RUB"},
		{name: "S5 zero", amount: 0, currency: "EUR", want: "0.00 EUR"},
		{name: "S6 typical", amount: 1999, currency: "EUR", want: "19.99 EUR"},
		{name: "S7 large", amount: 1000000, currency: "RUB", want: "10000.00 RUB"},
		// 101/100=1, 101%100=1; валюта печатается как пришла
		{name: "S8 currency as is", amount: 101, currency: "usd", want: "1.01 usd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := mustJSON(t, deposit(TEST_USER_2, tt.amount, tt.currency))

			got, err := Build(envelope(testEventID, models.EVENT_TRANSACTION_COMPLETED, payload))

			require.NoError(t, err)
			want := []models.Notification{
				note(testUser2, models.EVENT_TRANSACTION_COMPLETED, "Счёт A2 пополнен на "+tt.want+"."),
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestBuild_KYC(t *testing.T) {
	approved := models.EVENT_APPLICATION_APPROVED
	rejected := models.EVENT_APPLICATION_REJECTED
	docs := models.EVENT_DOCS_REQUESTED

	tests := []struct {
		name      string
		eventType string
		payload   func(t *testing.T) json.RawMessage
		want      []models.Notification
		wantErr   error
	}{
		{
			name:      "K1 approved notifies client",
			eventType: approved,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, application(TEST_USER_1, "approved", "", nil)) },
			want:      []models.Notification{note(testUser1, approved, "Поздравляем, вы верифицированы.")},
		},
		{
			name:      "K2 approved without user",
			eventType: approved,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, application("", "approved", "", nil)) },
			want:      nil,
		},
		{
			name:      "K3 approved with invalid user uuid",
			eventType: approved,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_BAD_UUID, "approved", "", nil))
			},
			wantErr: domain.ErrPoisonMessage,
		},
		{
			name:      "K4 approved with broken json",
			eventType: approved,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "K4a approved with empty payload",
			eventType: approved,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage{} },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "K5 auto rejected",
			eventType: rejected,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, models.APPLICATION_STATUS_AUTO_REJECTED, "sanctions match", nil))
			},
			want: []models.Notification{note(testUser1, rejected,
				"Заявка на верификацию не прошла автоматическую проверку: sanctions match. Исправьте анкету и подайте заявку снова.")},
		},
		{
			name:      "K6 rejected by officer",
			eventType: rejected,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, models.APPLICATION_STATUS_REJECTED, "fake documents", nil))
			},
			want: []models.Notification{note(testUser1, rejected, "Заявка на верификацию отклонена: fake documents.")},
		},
		{
			name:      "K7 rejected with empty status",
			eventType: rejected,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, application(TEST_USER_1, "", "reason", nil)) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "K8 rejected with approved status",
			eventType: rejected,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, "approved", "reason", nil))
			},
			wantErr: domain.ErrPoisonMessage,
		},
		{
			name:      "K9 rejected with status in other case",
			eventType: rejected,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, "Rejected", "reason", nil))
			},
			wantErr: domain.ErrPoisonMessage,
		},
		{
			name:      "K10 rejected with broken json",
			eventType: rejected,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "K11 rejected with invalid user uuid",
			eventType: rejected,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_BAD_UUID, models.APPLICATION_STATUS_REJECTED, "reason", nil))
			},
			wantErr: domain.ErrPoisonMessage,
		},
		{
			name:      "K12 rejected without user",
			eventType: rejected,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application("", models.APPLICATION_STATUS_REJECTED, "reason", nil))
			},
			want: nil,
		},
		{
			name:      "K13 docs requested with known and unknown types and reason",
			eventType: docs,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, "docs_requested", "скан нечитаем",
					[]string{"passport", "proof_of_address", "driver_license"}))
			},
			want: []models.Notification{note(testUser1, docs,
				"Для проверки нужны документы: паспорт, подтверждение адреса, driver_license. Комментарий сотрудника: скан нечитаем")},
		},
		{
			name:      "K14 docs requested single type",
			eventType: docs,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, "docs_requested", "плохое освещение", []string{"selfie"}))
			},
			want: []models.Notification{note(testUser1, docs,
				"Для проверки нужны документы: селфи. Комментарий сотрудника: плохое освещение")},
		},
		{
			name:      "K15 docs requested without reason keeps order",
			eventType: docs,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, "docs_requested", "", []string{"selfie", "passport"}))
			},
			want: []models.Notification{note(testUser1, docs, "Для проверки нужны документы: селфи, паспорт.")},
		},
		{
			name:      "K16 docs requested with broken json",
			eventType: docs,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
			wantErr:   domain.ErrPoisonMessage,
		},
		{
			name:      "K17 docs requested with wrong requested_types type",
			eventType: docs,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, map[string]any{"user_id": TEST_USER_1, "requested_types": "passport"})
			},
			wantErr: domain.ErrPoisonMessage,
		},
		{
			name:      "K18 docs requested with invalid user uuid",
			eventType: docs,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_BAD_UUID, "docs_requested", "", []string{"passport"}))
			},
			wantErr: domain.ErrPoisonMessage,
		},
		{
			name:      "K19 docs requested without user",
			eventType: docs,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application("", "docs_requested", "", []string{"passport"}))
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(envelope(testEventID, tt.eventType, tt.payload(t)))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, nonNil(tt.want), nonNil(got))
		})
	}
}

func TestBuild_Ignored(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   func(t *testing.T) json.RawMessage
	}{
		{
			name:      "I1 submitted with valid payload",
			eventType: models.EVENT_APPLICATION_SUBMITTED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, application(TEST_USER_1, "submitted", "", nil)) },
		},
		{
			name:      "I2 submitted with broken payload",
			eventType: models.EVENT_APPLICATION_SUBMITTED,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
		},
		{
			name:      "I3 unknown ledger event with valid payload",
			eventType: TEST_UNKNOWN_EVENT,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_USER_1, "", TEST_USER_2)) },
		},
		{
			name:      "I4 unknown kyc event with broken payload",
			eventType: TEST_UNKNOWN_KYC,
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
		},
		{
			name:      "I5 empty event type with broken payload",
			eventType: "",
			payload:   func(t *testing.T) json.RawMessage { return json.RawMessage(TEST_BROKEN_JSON) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(envelope(testEventID, tt.eventType, tt.payload(t)))

			require.NoError(t, err)
			assert.Empty(t, got)
		})
	}
}

func TestBuild_Invariants(t *testing.T) {
	invariantEventID := uuid.MustParse(TEST_INVARIANT_EVENT)

	tests := []struct {
		name      string
		eventType string
		payload   func(t *testing.T) json.RawMessage
	}{
		{
			name:      "deposit",
			eventType: models.EVENT_TRANSACTION_COMPLETED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, deposit(TEST_USER_2, 150050, "RUB")) },
		},
		{
			name:      "transfer between users",
			eventType: models.EVENT_TRANSACTION_COMPLETED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_USER_1, "", TEST_USER_2)) },
		},
		{
			name:      "transfer between own accounts",
			eventType: models.EVENT_TRANSACTION_COMPLETED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer(TEST_USER_1, "", TEST_USER_1)) },
		},
		{
			name:      "transfer with sender from initiator",
			eventType: models.EVENT_TRANSACTION_COMPLETED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, transfer("", TEST_USER_1, TEST_USER_2)) },
		},
		{
			name:      "declined",
			eventType: models.EVENT_TRANSACTION_DECLINED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, declined(TEST_USER_1, "", TEST_USER_2)) },
		},
		{
			name:      "approved",
			eventType: models.EVENT_APPLICATION_APPROVED,
			payload:   func(t *testing.T) json.RawMessage { return mustJSON(t, application(TEST_USER_1, "approved", "", nil)) },
		},
		{
			name:      "auto rejected",
			eventType: models.EVENT_APPLICATION_REJECTED,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, models.APPLICATION_STATUS_AUTO_REJECTED, "reason", nil))
			},
		},
		{
			name:      "rejected",
			eventType: models.EVENT_APPLICATION_REJECTED,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, models.APPLICATION_STATUS_REJECTED, "reason", nil))
			},
		},
		{
			name:      "docs requested",
			eventType: models.EVENT_DOCS_REQUESTED,
			payload: func(t *testing.T) json.RawMessage {
				return mustJSON(t, application(TEST_USER_1, "docs_requested", "reason",
					[]string{"passport", "selfie", "proof_of_address"}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(envelope(invariantEventID, tt.eventType, tt.payload(t)))

			require.NoError(t, err)
			seen := make(map[uuid.UUID]int, len(got))
			for _, notification := range got {
				assert.Equal(t, invariantEventID, notification.EventID)
				assert.Equal(t, tt.eventType, notification.EventType)
				assert.Equal(t, uuid.Nil, notification.ID)
				assert.True(t, notification.CreatedAt.IsZero())
				seen[notification.UserID]++
			}
			assert.Len(t, seen, len(got))
		})
	}
}
