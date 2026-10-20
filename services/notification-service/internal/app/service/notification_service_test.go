package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

const (
	TEST_EVENT_ID       = "0199f0a1-1111-7000-8000-000000000001"
	TEST_USER_ID        = "0199f0a1-2222-7000-8000-000000000002"
	TEST_FROM_USER_ID   = "0199f0a1-3333-7000-8000-000000000003"
	TEST_TO_USER_ID     = "0199f0a1-4444-7000-8000-000000000004"
	TEST_OCCURRED_AT    = "2026-10-01T12:00:00Z"
	TEST_EVENT_UNKNOWN  = "some.unknown"
	TEST_TOPIC          = "ledger.events"
	TEST_APPROVED_TEXT  = "Поздравляем, вы верифицированы."
	TEST_TRANSFER_FROM  = "Перевод 1500.50 RUB на счёт A2 выполнен."
	TEST_TRANSFER_TO    = "Поступление 1500.50 RUB на счёт A2 со счёта A1."
	TEST_APPROVED_PAYLD = `{"application_id":"app-1","user_id":"` + TEST_USER_ID + `","status":"approved"}`
	TEST_TRANSFER_PAYLD = `{"transaction_id":"tx-1","type":"transfer","from_account_id":"A1","to_account_id":"A2","amount":150050,"currency":"RUB","initiator_id":"` + TEST_FROM_USER_ID + `","from_user_id":"` + TEST_FROM_USER_ID + `","to_user_id":"` + TEST_TO_USER_ID + `"}`
	TEST_REFUND_PAYLD   = `{"transaction_id":"tx-1","type":"refund","from_account_id":"A1","to_account_id":"A2","amount":150050,"currency":"RUB","initiator_id":"` + TEST_FROM_USER_ID + `","from_user_id":"` + TEST_FROM_USER_ID + `","to_user_id":"` + TEST_TO_USER_ID + `"}`
	TEST_BAD_TO_PAYLD   = `{"transaction_id":"tx-1","type":"transfer","from_account_id":"A1","to_account_id":"A2","amount":150050,"currency":"RUB","initiator_id":"` + TEST_FROM_USER_ID + `","from_user_id":"` + TEST_FROM_USER_ID + `","to_user_id":"not-a-uuid"}`
	TEST_SUBMIT_PAYLD   = `{"application_id":"app-1","user_id":"` + TEST_USER_ID + `","status":"submitted"}`
)

// envelopeValue собирает компактный JSON-конверт события.
func envelopeValue(eventID, eventType, payload string) []byte {
	return fmt.Appendf(nil, `{"event_id":%q,"event_type":%q,"occurred_at":%q,"payload":%s}`, eventID, eventType, TEST_OCCURRED_AT, payload)
}

// testOccurredAt - момент TEST_OCCURRED_AT.
func testOccurredAt(t *testing.T) time.Time {
	t.Helper()
	occurredAt, err := time.Parse(time.RFC3339, TEST_OCCURRED_AT)
	require.NoError(t, err)
	return occurredAt
}

// approvedNotifications - уведомления шаблона approved без ID и CreatedAt.
func approvedNotifications() []models.Notification {
	return []models.Notification{{
		UserID:    uuid.MustParse(TEST_USER_ID),
		EventID:   uuid.MustParse(TEST_EVENT_ID),
		EventType: models.EVENT_APPLICATION_APPROVED,
		Text:      TEST_APPROVED_TEXT,
	}}
}

// transferNotifications - уведомления шаблона перевода без ID и CreatedAt.
func transferNotifications() []models.Notification {
	return []models.Notification{
		{
			UserID:    uuid.MustParse(TEST_FROM_USER_ID),
			EventID:   uuid.MustParse(TEST_EVENT_ID),
			EventType: models.EVENT_TRANSACTION_COMPLETED,
			Text:      TEST_TRANSFER_FROM,
		},
		{
			UserID:    uuid.MustParse(TEST_TO_USER_ID),
			EventID:   uuid.MustParse(TEST_EVENT_ID),
			EventType: models.EVENT_TRANSACTION_COMPLETED,
			Text:      TEST_TRANSFER_TO,
		},
	}
}

// assertEnvelope сравнивает конверт целиком, момент OccurredAt - через Equal.
func assertEnvelope(t *testing.T, expected, actual models.Envelope) {
	t.Helper()
	assert.True(t, expected.OccurredAt.Equal(actual.OccurredAt), "occurred_at: expected %v, got %v", expected.OccurredAt, actual.OccurredAt)
	actual.OccurredAt = expected.OccurredAt
	assert.Equal(t, expected, actual)
}

// assertGeneratedFields проверяет ID и CreatedAt уведомлений по правилам H3.
func assertGeneratedFields(t *testing.T, notifications []models.Notification, before, after time.Time) {
	t.Helper()
	require.NotEmpty(t, notifications)
	createdAt := notifications[0].CreatedAt
	seen := make(map[uuid.UUID]struct{}, len(notifications))
	for _, notification := range notifications {
		assert.NotEqual(t, uuid.Nil, notification.ID)
		assert.Equal(t, uuid.Version(7), notification.ID.Version())
		seen[notification.ID] = struct{}{}
		assert.True(t, createdAt.Equal(notification.CreatedAt), "created_at differs: %v vs %v", createdAt, notification.CreatedAt)
		assert.Equal(t, time.UTC, notification.CreatedAt.Location())
	}
	assert.Len(t, seen, len(notifications))
	assert.False(t, createdAt.Before(before), "created_at %v is before %v", createdAt, before)
	assert.False(t, createdAt.After(after), "created_at %v is after %v", createdAt, after)
}

// withGenerated переносит ID и CreatedAt из фактических уведомлений в ожидаемые.
func withGenerated(t *testing.T, expected, actual []models.Notification) []models.Notification {
	t.Helper()
	require.Len(t, actual, len(expected))
	result := append([]models.Notification(nil), expected...)
	for i := range result {
		result[i].ID = actual[i].ID
		result[i].CreatedAt = actual[i].CreatedAt
	}
	return result
}

func TestHandleEvent_Poison(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
	}{
		{name: "nil value", value: nil},
		{name: "empty value", value: []byte{}},
		{name: "not json", value: []byte("not json")},
		{name: "json array", value: []byte(`[1,2]`)},
		{name: "event_id not uuid", value: envelopeValue("abc", models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD)},
		{name: "event_id missing", value: []byte(`{"event_type":"` + models.EVENT_APPLICATION_APPROVED + `","occurred_at":"` + TEST_OCCURRED_AT + `","payload":` + TEST_APPROVED_PAYLD + `}`)},
		{name: "event_id nil uuid", value: envelopeValue(uuid.Nil.String(), models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD)},
		{name: "event_type empty", value: envelopeValue(TEST_EVENT_ID, "", TEST_APPROVED_PAYLD)},
		{name: "event_type missing", value: []byte(`{"event_id":"` + TEST_EVENT_ID + `","occurred_at":"` + TEST_OCCURRED_AT + `","payload":` + TEST_APPROVED_PAYLD + `}`)},
		{name: "occurred_at not rfc3339", value: []byte(`{"event_id":"` + TEST_EVENT_ID + `","event_type":"` + models.EVENT_APPLICATION_APPROVED + `","occurred_at":"yesterday","payload":` + TEST_APPROVED_PAYLD + `}`)},
		{name: "template rejects refund", value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_REFUND_PAYLD)},
		{name: "template rejects bad to_user_id", value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_BAD_TO_PAYLD)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{}
			sender := &fakeSender{}
			svc := NewNotificationService(repo, sender)

			err := svc.HandleEvent(t.Context(), models.Message{Topic: TEST_TOPIC, Value: tt.value})

			require.ErrorIs(t, err, domain.ErrPoisonMessage)
			assert.Empty(t, repo.saveCalls)
			assert.Empty(t, sender.sendCalls)
		})
	}
}

func TestHandleEvent_Delivered(t *testing.T) {
	tests := []struct {
		name              string
		value             []byte
		wantEnvelope      func(t *testing.T) models.Envelope
		wantNotifications []models.Notification
	}{
		{
			name:  "approved",
			value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD),
			wantEnvelope: func(t *testing.T) models.Envelope {
				return models.Envelope{
					EventID:    uuid.MustParse(TEST_EVENT_ID),
					EventType:  models.EVENT_APPLICATION_APPROVED,
					OccurredAt: testOccurredAt(t),
					Payload:    json.RawMessage(TEST_APPROVED_PAYLD),
				}
			},
			wantNotifications: approvedNotifications(),
		},
		{
			name:  "transfer two recipients",
			value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_TRANSFER_PAYLD),
			wantEnvelope: func(t *testing.T) models.Envelope {
				return models.Envelope{
					EventID:    uuid.MustParse(TEST_EVENT_ID),
					EventType:  models.EVENT_TRANSACTION_COMPLETED,
					OccurredAt: testOccurredAt(t),
					Payload:    json.RawMessage(TEST_TRANSFER_PAYLD),
				}
			},
			wantNotifications: transferNotifications(),
		},
		{
			name:  "approved without occurred_at",
			value: []byte(`{"event_id":"` + TEST_EVENT_ID + `","event_type":"` + models.EVENT_APPLICATION_APPROVED + `","payload":` + TEST_APPROVED_PAYLD + `}`),
			wantEnvelope: func(t *testing.T) models.Envelope {
				// отсутствующий occurred_at даёт нулевое время
				return models.Envelope{
					EventID:   uuid.MustParse(TEST_EVENT_ID),
					EventType: models.EVENT_APPLICATION_APPROVED,
					Payload:   json.RawMessage(TEST_APPROVED_PAYLD),
				}
			},
			wantNotifications: approvedNotifications(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{saveResults: []bool{true}}
			sender := &fakeSender{}
			svc := NewNotificationService(repo, sender)

			before := time.Now()
			err := svc.HandleEvent(t.Context(), models.Message{Topic: TEST_TOPIC, Value: tt.value})
			after := time.Now()

			require.NoError(t, err)
			require.Len(t, repo.saveCalls, 1)
			saved := repo.saveCalls[0]
			assertEnvelope(t, tt.wantEnvelope(t), saved.envelope)
			assertGeneratedFields(t, saved.notifications, before, after)
			assert.Equal(t, withGenerated(t, tt.wantNotifications, saved.notifications), saved.notifications)
			assert.Equal(t, saved.notifications, sender.sendCalls)
		})
	}
}

func TestHandleEvent_DuplicateNotSent(t *testing.T) {
	repo := &fakeNotificationRepository{saveResults: []bool{false}}
	sender := &fakeSender{}
	svc := NewNotificationService(repo, sender)

	err := svc.HandleEvent(t.Context(), models.Message{
		Topic: TEST_TOPIC,
		Value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_TRANSFER_PAYLD),
	})

	require.NoError(t, err)
	assert.Len(t, repo.saveCalls, 1)
	assert.Empty(t, sender.sendCalls)
}

func TestHandleEvent_RepeatedMessageSentOnce(t *testing.T) {
	repo := &fakeNotificationRepository{saveResults: []bool{true, false}}
	sender := &fakeSender{}
	svc := NewNotificationService(repo, sender)
	msg := models.Message{
		Topic: TEST_TOPIC,
		Value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD),
	}
	wantEnvelope := models.Envelope{
		EventID:    uuid.MustParse(TEST_EVENT_ID),
		EventType:  models.EVENT_APPLICATION_APPROVED,
		OccurredAt: testOccurredAt(t),
		Payload:    json.RawMessage(TEST_APPROVED_PAYLD),
	}

	before := time.Now()
	require.NoError(t, svc.HandleEvent(t.Context(), msg))
	require.NoError(t, svc.HandleEvent(t.Context(), msg))
	after := time.Now()

	require.Len(t, repo.saveCalls, 2)
	first := repo.saveCalls[0]
	second := repo.saveCalls[1]
	assertEnvelope(t, wantEnvelope, first.envelope)
	assertEnvelope(t, wantEnvelope, second.envelope)
	assertGeneratedFields(t, first.notifications, before, after)
	assertGeneratedFields(t, second.notifications, before, after)
	assert.Equal(t, withGenerated(t, approvedNotifications(), first.notifications), first.notifications)
	assert.Equal(t, withGenerated(t, approvedNotifications(), second.notifications), second.notifications)
	assert.Equal(t, first.notifications, sender.sendCalls)
}

func TestHandleEvent_SaveError(t *testing.T) {
	saveErr := errors.New("db is down")
	repo := &fakeNotificationRepository{saveErr: saveErr}
	sender := &fakeSender{}
	svc := NewNotificationService(repo, sender)

	err := svc.HandleEvent(t.Context(), models.Message{
		Topic: TEST_TOPIC,
		Value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD),
	})

	require.ErrorIs(t, err, saveErr)
	assert.NotErrorIs(t, err, domain.ErrPoisonMessage)
	assert.Len(t, repo.saveCalls, 1)
	assert.Empty(t, sender.sendCalls)
}

func TestHandleEvent_SendError(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
	}{
		{name: "approved", value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD)},
		{name: "transfer", value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_TRANSFER_PAYLD)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendErr := errors.New("delivery failed")
			repo := &fakeNotificationRepository{saveResults: []bool{true}}
			sender := &fakeSender{sendErr: sendErr}
			svc := NewNotificationService(repo, sender)

			err := svc.HandleEvent(t.Context(), models.Message{Topic: TEST_TOPIC, Value: tt.value})

			require.ErrorIs(t, err, sendErr)
			assert.NotErrorIs(t, err, domain.ErrPoisonMessage)
		})
	}
}

func TestHandleEvent_NoNotifications(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
	}{
		{name: "application submitted", value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_SUBMITTED, TEST_SUBMIT_PAYLD)},
		{name: "unknown event type", value: envelopeValue(TEST_EVENT_ID, TEST_EVENT_UNKNOWN, `{}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{saveResults: []bool{true}}
			sender := &fakeSender{}
			svc := NewNotificationService(repo, sender)

			err := svc.HandleEvent(t.Context(), models.Message{Topic: TEST_TOPIC, Value: tt.value})

			require.NoError(t, err)
			assert.Empty(t, sender.sendCalls)
		})
	}
}

// Инвариант: Send вызывается только после Save, вернувшего true.
func TestHandleEvent_InvariantSendOnlyAfterSave(t *testing.T) {
	tests := []struct {
		name        string
		value       []byte
		saveResults []bool
		saveErr     error
		wantJournal []string
	}{
		{
			name:        "approved saved",
			value:       envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD),
			saveResults: []bool{true},
			wantJournal: []string{"save", "send"},
		},
		{
			name:        "transfer saved",
			value:       envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_TRANSFER_PAYLD),
			saveResults: []bool{true},
			wantJournal: []string{"save", "send", "send"},
		},
		{
			name:        "transfer duplicate",
			value:       envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_TRANSFER_PAYLD),
			saveResults: []bool{false},
			wantJournal: []string{"save"},
		},
		{
			name:        "approved save error",
			value:       envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD),
			saveErr:     errors.New("db is down"),
			wantJournal: []string{"save"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			journal := []string{}
			repo := &fakeNotificationRepository{saveResults: tt.saveResults, saveErr: tt.saveErr, journal: &journal}
			sender := &fakeSender{journal: &journal}
			svc := NewNotificationService(repo, sender)

			// результат проверяется в других тестах, здесь важен только порядок вызовов
			err := svc.HandleEvent(t.Context(), models.Message{Topic: TEST_TOPIC, Value: tt.value})

			assert.ErrorIs(t, err, tt.saveErr)
			assert.Equal(t, tt.wantJournal, journal)
		})
	}
}

// Инвариант: ошибка конверта или шаблона - poison, ошибка зависимостей - нет.
func TestHandleEvent_InvariantPoisonClassification(t *testing.T) {
	saveErr := errors.New("db is down")
	sendErr := errors.New("delivery failed")
	tests := []struct {
		name       string
		value      []byte
		saveErr    error
		sendErr    error
		wantErr    error
		wantPoison bool
	}{
		{name: "not json", value: []byte("not json"), wantErr: domain.ErrPoisonMessage, wantPoison: true},
		{name: "nil event_id", value: envelopeValue(uuid.Nil.String(), models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD), wantErr: domain.ErrPoisonMessage, wantPoison: true},
		{name: "refund", value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_REFUND_PAYLD), wantErr: domain.ErrPoisonMessage, wantPoison: true},
		{name: "bad to_user_id", value: envelopeValue(TEST_EVENT_ID, models.EVENT_TRANSACTION_COMPLETED, TEST_BAD_TO_PAYLD), wantErr: domain.ErrPoisonMessage, wantPoison: true},
		{name: "save error", value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD), saveErr: saveErr, wantErr: saveErr, wantPoison: false},
		{name: "send error", value: envelopeValue(TEST_EVENT_ID, models.EVENT_APPLICATION_APPROVED, TEST_APPROVED_PAYLD), sendErr: sendErr, wantErr: sendErr, wantPoison: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{saveResults: []bool{true}, saveErr: tt.saveErr}
			sender := &fakeSender{sendErr: tt.sendErr}
			svc := NewNotificationService(repo, sender)

			err := svc.HandleEvent(t.Context(), models.Message{Topic: TEST_TOPIC, Value: tt.value})

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantPoison, errors.Is(err, domain.ErrPoisonMessage))
		})
	}
}

func TestListNotifications_Page(t *testing.T) {
	tests := []struct {
		name     string
		page     models.Page
		wantPage models.Page
	}{
		{name: "limit zero", page: models.Page{Limit: 0, Offset: 0}, wantPage: models.Page{Limit: domain.DEFAULT_PAGE_SIZE, Offset: 0}},
		{name: "limit negative", page: models.Page{Limit: -5, Offset: 0}, wantPage: models.Page{Limit: domain.DEFAULT_PAGE_SIZE, Offset: 0}},
		{name: "limit one", page: models.Page{Limit: 1, Offset: 0}, wantPage: models.Page{Limit: 1, Offset: 0}},
		{name: "limit twenty", page: models.Page{Limit: 20, Offset: 0}, wantPage: models.Page{Limit: 20, Offset: 0}},
		{name: "limit max", page: models.Page{Limit: 100, Offset: 0}, wantPage: models.Page{Limit: 100, Offset: 0}},
		{name: "limit above max", page: models.Page{Limit: 101, Offset: 0}, wantPage: models.Page{Limit: domain.MAX_PAGE_SIZE, Offset: 0}},
		{name: "limit far above max", page: models.Page{Limit: 1000, Offset: 0}, wantPage: models.Page{Limit: domain.MAX_PAGE_SIZE, Offset: 0}},
		{name: "offset negative", page: models.Page{Limit: 10, Offset: -1}, wantPage: models.Page{Limit: 10, Offset: 0}},
		{name: "offset zero", page: models.Page{Limit: 10, Offset: 0}, wantPage: models.Page{Limit: 10, Offset: 0}},
		{name: "offset positive", page: models.Page{Limit: 10, Offset: 40}, wantPage: models.Page{Limit: 10, Offset: 40}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.MustParse(TEST_USER_ID)
			repo := &fakeNotificationRepository{listResult: []*models.Notification{}}
			svc := NewNotificationService(repo, &fakeSender{})

			_, err := svc.ListNotifications(t.Context(), userID, tt.page)

			require.NoError(t, err)
			assert.Equal(t, []listCall{{userID: userID, page: tt.wantPage}}, repo.listCalls)
		})
	}
}

func TestListNotifications_Result(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	tests := []struct {
		name       string
		listResult []*models.Notification
	}{
		{
			name: "two notifications",
			listResult: []*models.Notification{
				{ID: uuid.MustParse("0199f0a1-5555-7000-8000-000000000005"), UserID: userID, EventID: uuid.MustParse(TEST_EVENT_ID), EventType: models.EVENT_APPLICATION_APPROVED, Text: TEST_APPROVED_TEXT, CreatedAt: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)},
				{ID: uuid.MustParse("0199f0a1-6666-7000-8000-000000000006"), UserID: userID, EventID: uuid.MustParse(TEST_EVENT_ID), EventType: models.EVENT_TRANSACTION_COMPLETED, Text: TEST_TRANSFER_TO, CreatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
		{name: "empty slice", listResult: []*models.Notification{}},
		{name: "nil slice", listResult: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{listResult: tt.listResult}
			svc := NewNotificationService(repo, &fakeSender{})

			got, err := svc.ListNotifications(t.Context(), userID, models.Page{Limit: 10, Offset: 0})

			require.NoError(t, err)
			// Equal различает nil и пустой срез
			assert.Equal(t, tt.listResult, got)
			require.Len(t, got, len(tt.listResult))
			for i := range tt.listResult {
				assert.Same(t, tt.listResult[i], got[i])
			}
		})
	}
}

func TestListNotifications_RepoError(t *testing.T) {
	repoErr := errors.New("db is down")
	repo := &fakeNotificationRepository{listErr: repoErr}
	svc := NewNotificationService(repo, &fakeSender{})

	_, err := svc.ListNotifications(t.Context(), uuid.MustParse(TEST_USER_ID), models.Page{Limit: 10, Offset: 0})

	require.ErrorIs(t, err, repoErr)
}
