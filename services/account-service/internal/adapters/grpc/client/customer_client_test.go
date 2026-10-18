package client

import (
	"errors"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/customer/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TEST_USER_ID  = "11111111-2222-3333-4444-555555555555"
	TEST_NIL_UUID = "00000000-0000-0000-0000-000000000000"
)

func TestCustomerClient_IsActive(t *testing.T) {
	tests := []struct {
		name       string
		userID     uuid.UUID
		status     customerv1.CustomerStatus
		want       bool
		wantUserID string
	}{
		{name: "active", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_ACTIVE, want: true, wantUserID: TEST_USER_ID},
		{name: "unspecified", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_UNSPECIFIED, want: false, wantUserID: TEST_USER_ID},
		{name: "new", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_NEW, want: false, wantUserID: TEST_USER_ID},
		{name: "profile filled", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_PROFILE_FILLED, want: false, wantUserID: TEST_USER_ID},
		{name: "on kyc", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_ON_KYC, want: false, wantUserID: TEST_USER_ID},
		{name: "rejected", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_REJECTED, want: false, wantUserID: TEST_USER_ID},
		{name: "blocked", userID: uuid.MustParse(TEST_USER_ID), status: customerv1.CustomerStatus_BLOCKED, want: false, wantUserID: TEST_USER_ID},
		{name: "nil user id active", userID: uuid.Nil, status: customerv1.CustomerStatus_ACTIVE, want: true, wantUserID: TEST_NIL_UUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeCustomerServiceClient{status: tt.status}
			customerClient := NewCustomerClient(fake)

			got, err := customerClient.IsActive(t.Context(), tt.userID)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, []string{tt.wantUserID}, fake.userIDs)
		})
	}
}

func TestCustomerClient_IsActive_NotFound(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{name: "standard message", message: "customer not found"},
		{name: "empty message", message: ""},
		{name: "arbitrary message", message: "whatever"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeCustomerServiceClient{getStatusErr: status.Error(codes.NotFound, tt.message)}
			customerClient := NewCustomerClient(fake)

			got, err := customerClient.IsActive(t.Context(), uuid.MustParse(TEST_USER_ID))

			require.ErrorIs(t, err, domain.ErrCustomerNotFound)
			assert.False(t, got)
			assert.Equal(t, []string{TEST_USER_ID}, fake.userIDs)
		})
	}
}

func TestCustomerClient_IsActive_UpstreamError(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		wantCode codes.Code
	}{
		{name: "unavailable", upstream: status.Error(codes.Unavailable, "customer-service down"), wantCode: codes.Unavailable},
		{name: "internal", upstream: status.Error(codes.Internal, "internal"), wantCode: codes.Internal},
		{name: "deadline exceeded", upstream: status.Error(codes.DeadlineExceeded, "deadline"), wantCode: codes.DeadlineExceeded},
		{name: "permission denied", upstream: status.Error(codes.PermissionDenied, "denied"), wantCode: codes.PermissionDenied},
		// обычная ошибка без статуса даёт codes.Unknown
		{name: "plain error", upstream: errors.New("boom"), wantCode: codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeCustomerServiceClient{getStatusErr: tt.upstream}
			customerClient := NewCustomerClient(fake)

			got, err := customerClient.IsActive(t.Context(), uuid.MustParse(TEST_USER_ID))

			require.ErrorIs(t, err, tt.upstream)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.NotErrorIs(t, err, domain.ErrCustomerNotFound)
			assert.False(t, got)
			assert.Equal(t, []string{TEST_USER_ID}, fake.userIDs)
		})
	}
}
