package server

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

const (
	TEST_USER_ID           = "11111111-1111-4111-8111-111111111111"
	TEST_USER_ID_UPPER     = "11111111-1111-4111-8111-11111111111A"
	TEST_USER_ID_LOWER     = "11111111-1111-4111-8111-11111111111a"
	TEST_NOTIFICATION_ID_A = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	TEST_NOTIFICATION_ID_B = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	TEST_NOTIFICATION_ID_C = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	TEST_EVENT_ID_E        = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	TEST_EVENT_ID_F        = "ffffffff-ffff-4fff-8fff-ffffffffffff"
	TEST_EVENT_TYPE_KYC    = "kyc.application_approved"
	TEST_EVENT_TYPE_LEDGER = "ledger.transaction_completed"
	TEST_TEXT_KYC          = "Поздравляем, вы верифицированы."
	TEST_TEXT_LEDGER       = "Перевод проведён."
	TEST_DEFAULT_LIMIT     = 20
	TEST_MAX_LIMIT         = 100
)

// testCreatedAt - момент создания уведомления из примера спецификации
func testCreatedAt() time.Time {
	return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
}

// newDomainNotification собирает доменное уведомление клиента TEST_USER_ID
func newDomainNotification(notificationID, eventID, eventType, text string, createdAt time.Time) *models.Notification {
	return &models.Notification{
		ID:        uuid.MustParse(notificationID),
		UserID:    uuid.MustParse(TEST_USER_ID),
		EventID:   uuid.MustParse(eventID),
		EventType: eventType,
		Text:      text,
		CreatedAt: createdAt,
	}
}

// newProtoNotification собирает ожидаемое proto-уведомление клиента TEST_USER_ID
func newProtoNotification(notificationID, eventID, eventType, text string, createdAt time.Time) *notificationv1.Notification {
	return &notificationv1.Notification{
		Id:        notificationID,
		UserId:    TEST_USER_ID,
		EventId:   eventID,
		EventType: eventType,
		Text:      text,
		CreatedAt: timestamppb.New(createdAt),
	}
}

// newTestServer собирает сервер на настоящем сервисном слое и фейковом репозитории
func newTestServer(repo *fakeNotificationRepository) *NotificationServer {
	svc := &service.Service{Notifications: service.NewNotificationService(repo, &fakeSender{})}
	return NewNotificationServer(svc, slog.New(slog.DiscardHandler))
}

func TestNotificationServer_ListNotifications(t *testing.T) {
	repoErr := errors.New("db is down")
	userID := uuid.MustParse(TEST_USER_ID)
	// 15:00 по +03:00 - тот же момент, что 12:00 UTC
	createdAtMoscow := time.Date(2026, 10, 1, 15, 0, 0, 0, time.FixedZone("MSK", 3*60*60))

	tests := []struct {
		name       string
		userID     string
		limit      int32
		offset     int32
		listResult []*models.Notification
		listErr    error
		want       []*notificationv1.Notification
		wantCalls  []listByUserCall
		wantErr    error
	}{
		{
			name:    "empty user_id is rejected",
			userID:  "",
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "non uuid user_id is rejected",
			userID:  "not-a-uuid",
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "numeric user_id is rejected",
			userID:  "123",
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "user_id one char shorter than uuid is rejected",
			userID:  "11111111-1111-4111-8111-11111111111",
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:       "zero limit becomes default and nil result gives empty list",
			userID:     TEST_USER_ID,
			limit:      0,
			offset:     0,
			listResult: nil,
			want:       nil,
			wantCalls:  []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
		},
		{
			name:      "limit and offset are passed as is",
			userID:    TEST_USER_ID,
			limit:     5,
			offset:    10,
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: 5, Offset: 10}}},
		},
		{
			name:      "limit above max is capped",
			userID:    TEST_USER_ID,
			limit:     500,
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_MAX_LIMIT, Offset: 0}}},
		},
		{
			name:      "limit equal to max is kept",
			userID:    TEST_USER_ID,
			limit:     100,
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: 100, Offset: 0}}},
		},
		{
			name:      "limit one above max is capped",
			userID:    TEST_USER_ID,
			limit:     101,
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_MAX_LIMIT, Offset: 0}}},
		},
		{
			name:      "limit one is kept",
			userID:    TEST_USER_ID,
			limit:     1,
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: 1, Offset: 0}}},
		},
		{
			name:      "uppercase user_id is accepted",
			userID:    TEST_USER_ID_UPPER,
			wantCalls: []listByUserCall{{userID: uuid.MustParse(TEST_USER_ID_LOWER), page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
		},
		{
			name:   "single notification is mapped to proto",
			userID: TEST_USER_ID,
			listResult: []*models.Notification{
				newDomainNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
			},
			want: []*notificationv1.Notification{
				newProtoNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
			},
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
		},
		{
			name:   "two notifications keep order",
			userID: TEST_USER_ID,
			listResult: []*models.Notification{
				newDomainNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
				newDomainNotification(TEST_NOTIFICATION_ID_B, TEST_EVENT_ID_F, TEST_EVENT_TYPE_LEDGER, TEST_TEXT_LEDGER, testCreatedAt().Add(time.Hour)),
			},
			want: []*notificationv1.Notification{
				newProtoNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
				newProtoNotification(TEST_NOTIFICATION_ID_B, TEST_EVENT_ID_F, TEST_EVENT_TYPE_LEDGER, TEST_TEXT_LEDGER, testCreatedAt().Add(time.Hour)),
			},
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
		},
		{
			name:   "created_at in non utc zone keeps the moment",
			userID: TEST_USER_ID,
			listResult: []*models.Notification{
				newDomainNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, createdAtMoscow),
			},
			want: []*notificationv1.Notification{
				newProtoNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
			},
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
		},
		{
			name:       "empty non nil result gives empty list",
			userID:     TEST_USER_ID,
			listResult: []*models.Notification{},
			want:       nil,
			wantCalls:  []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
		},
		{
			name:      "repository error is hidden behind internal",
			userID:    TEST_USER_ID,
			listErr:   repoErr,
			wantCalls: []listByUserCall{{userID: userID, page: models.Page{Limit: TEST_DEFAULT_LIMIT, Offset: 0}}},
			wantErr:   domain.ErrListFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{listResult: tt.listResult, listErr: tt.listErr}
			srv := newTestServer(repo)

			resp, err := srv.ListNotifications(t.Context(), &notificationv1.ListNotificationsRequest{
				UserId: tt.userID,
				Limit:  tt.limit,
				Offset: tt.offset,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				// исходная ошибка репозитория наружу не уходит (для S1 listErr == nil, errors.Is(err, nil) == false)
				assert.NotErrorIs(t, err, tt.listErr)
				assert.Nil(t, resp)
				assert.Equal(t, tt.wantCalls, repo.calls())
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Notifications, len(tt.want))
			for i := range tt.want {
				assert.Truef(t, proto.Equal(tt.want[i], resp.Notifications[i]),
					"notification %d mismatch:\nwant: %v\ngot:  %v", i, tt.want[i], resp.Notifications[i])
			}
			assert.Equal(t, tt.wantCalls, repo.calls())
		})
	}
}

func TestNotificationServer_ListNotifications_LengthInvariant(t *testing.T) {
	tests := []struct {
		name       string
		listResult []*models.Notification
	}{
		{
			name:       "nil result",
			listResult: nil,
		},
		{
			name: "one notification",
			listResult: []*models.Notification{
				newDomainNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
			},
		},
		{
			name: "three notifications",
			listResult: []*models.Notification{
				newDomainNotification(TEST_NOTIFICATION_ID_A, TEST_EVENT_ID_E, TEST_EVENT_TYPE_KYC, TEST_TEXT_KYC, testCreatedAt()),
				newDomainNotification(TEST_NOTIFICATION_ID_B, TEST_EVENT_ID_F, TEST_EVENT_TYPE_LEDGER, TEST_TEXT_LEDGER, testCreatedAt()),
				newDomainNotification(TEST_NOTIFICATION_ID_C, TEST_EVENT_ID_F, TEST_EVENT_TYPE_LEDGER, TEST_TEXT_LEDGER, testCreatedAt()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeNotificationRepository{listResult: tt.listResult}
			srv := newTestServer(repo)

			resp, err := srv.ListNotifications(t.Context(), &notificationv1.ListNotificationsRequest{UserId: TEST_USER_ID})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, len(tt.listResult), len(resp.Notifications))
		})
	}
}
