package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

func TestSetStatus_BadIDs(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		actorID string
		wantErr error
	}{
		{name: "плохой user_id", userID: "bad", actorID: TEST_ACTOR_ID, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "плохой actor_id", userID: TEST_USER_ID, actorID: "bad", wantErr: domain.ErrActorIDMustBeUUID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)

			_, err := env.statuses.SetStatus(context.Background(), &customerv1.SetStatusRequest{
				UserId:  tc.userID,
				Status:  customerv1.CustomerStatus_ACTIVE,
				ActorId: tc.actorID,
			})

			requireStatusErr(t, err, codes.InvalidArgument, tc.wantErr)
		})
	}
}

func TestSetStatus_ActorID(t *testing.T) {
	actorID := testActorID

	tests := []struct {
		name        string
		rawActorID  string
		wantActorID *uuid.UUID
	}{
		{name: "системный переход", rawActorID: "", wantActorID: nil},
		{name: "инициатор задан", rawActorID: TEST_ACTOR_ID, wantActorID: &actorID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(models.STATUS_ON_KYC)
			change := &models.StatusChange{Previous: models.STATUS_ON_KYC, Current: models.STATUS_ACTIVE, ChangedAt: testTime}
			env.repo.On("SetStatus", mock.Anything, testUserID, models.STATUS_ACTIVE, "", tc.wantActorID, []models.Status{models.STATUS_ON_KYC}).
				Return(change, nil).Once()

			_, err := env.statuses.SetStatus(context.Background(), &customerv1.SetStatusRequest{
				UserId:  TEST_USER_ID,
				Status:  customerv1.CustomerStatus_ACTIVE,
				ActorId: tc.rawActorID,
			})

			require.NoError(t, err)
		})
	}
}

func TestSetStatus_Errors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(env *testEnv)
		target   customerv1.CustomerStatus
		reason   string
		wantCode codes.Code
		wantErr  error
	}{
		{
			name: "клиент не найден",
			setup: func(env *testEnv) {
				env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(nil, nil, domain.ErrCustomerNotFound).Once()
			},
			target:   customerv1.CustomerStatus_ACTIVE,
			wantCode: codes.NotFound,
			wantErr:  domain.ErrCustomerMissing,
		},
		{
			name: "статус сменился",
			setup: func(env *testEnv) {
				env.expectStatus(models.STATUS_ON_KYC)
				env.repo.On("SetStatus", mock.Anything, testUserID, models.STATUS_ACTIVE, "", (*uuid.UUID)(nil), []models.Status{models.STATUS_ON_KYC}).
					Return(nil, domain.ErrStatusConflict).Once()
			},
			target:   customerv1.CustomerStatus_ACTIVE,
			wantCode: codes.Aborted,
			wantErr:  domain.ErrStatusChanged,
		},
		{
			name:     "блокировка",
			setup:    func(env *testEnv) { env.expectStatus(models.STATUS_ON_KYC) },
			target:   customerv1.CustomerStatus_BLOCKED,
			wantCode: codes.Unimplemented,
			wantErr:  domain.ErrBlockingNotImplemented,
		},
		{
			name:     "отказ без причины",
			setup:    func(env *testEnv) { env.expectStatus(models.STATUS_ON_KYC) },
			target:   customerv1.CustomerStatus_REJECTED,
			wantCode: codes.InvalidArgument,
			wantErr:  domain.ErrReasonRequired,
		},
		{
			name: "сбой базы",
			setup: func(env *testEnv) {
				env.expectStatus(models.STATUS_ON_KYC)
				env.repo.On("SetStatus", mock.Anything, testUserID, models.STATUS_ACTIVE, "", (*uuid.UUID)(nil), []models.Status{models.STATUS_ON_KYC}).
					Return(nil, errDB).Once()
			},
			target:   customerv1.CustomerStatus_ACTIVE,
			wantCode: codes.Internal,
			wantErr:  domain.ErrSetStatusFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			tc.setup(env)

			_, err := env.statuses.SetStatus(context.Background(), &customerv1.SetStatusRequest{
				UserId: TEST_USER_ID,
				Status: tc.target,
				Reason: tc.reason,
			})

			requireStatusErr(t, err, tc.wantCode, tc.wantErr)
		})
	}
}

func TestSetStatus_TransitionForbidden(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)

	_, err := env.statuses.SetStatus(context.Background(), &customerv1.SetStatusRequest{
		UserId: TEST_USER_ID,
		Status: customerv1.CustomerStatus_ACTIVE,
	})

	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	// Наружу уходит текст без подробностей перехода.
	assert.Equal(t, status.Convert(domain.ErrStatusTransitionForbidden).Message(), status.Convert(err).Message())
	assert.NotContains(t, err.Error(), "new -> active")
}

func TestSetStatus_Success(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_ON_KYC)
	change := &models.StatusChange{Previous: models.STATUS_ON_KYC, Current: models.STATUS_ACTIVE, ChangedAt: testTime}
	env.repo.On("SetStatus", mock.Anything, testUserID, models.STATUS_ACTIVE, "", (*uuid.UUID)(nil), []models.Status{models.STATUS_ON_KYC}).
		Return(change, nil).Once()

	got, err := env.statuses.SetStatus(context.Background(), &customerv1.SetStatusRequest{
		UserId: TEST_USER_ID,
		Status: customerv1.CustomerStatus_ACTIVE,
	})

	require.NoError(t, err)
	assert.Equal(t, customerv1.CustomerStatus_ACTIVE, got.Status)
	assert.Equal(t, customerv1.CustomerStatus_ON_KYC, got.PreviousStatus)
	assert.Equal(t, testTime, got.StatusChangedAt.AsTime())
}

func TestSetStatus_Repeat(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_ACTIVE)

	got, err := env.statuses.SetStatus(context.Background(), &customerv1.SetStatusRequest{
		UserId: TEST_USER_ID,
		Status: customerv1.CustomerStatus_ACTIVE,
	})

	require.NoError(t, err)
	assert.Equal(t, customerv1.CustomerStatus_ACTIVE, got.Status)
	assert.Equal(t, customerv1.CustomerStatus_ACTIVE, got.PreviousStatus)
	assert.Equal(t, testTime, got.StatusChangedAt.AsTime())
	env.repo.AssertNotCalled(t, "SetStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestParseActorID(t *testing.T) {
	t.Run("пусто", func(t *testing.T) {
		got, err := parseActorID("")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("валидный", func(t *testing.T) {
		got, err := parseActorID(TEST_ACTOR_ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, testActorID, *got)
	})

	t.Run("плохой", func(t *testing.T) {
		_, err := parseActorID("bad")
		require.Error(t, err)
	})
}
