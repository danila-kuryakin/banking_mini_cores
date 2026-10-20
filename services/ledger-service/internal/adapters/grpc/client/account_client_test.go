package client

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/account/v1"
)

const (
	TEST_ACCOUNT_ID       = "5a0c2b7e-1d3f-4e6a-8b9c-0d1e2f3a4b01"
	TEST_OTHER_ACCOUNT_ID = "5a0c2b7e-1d3f-4e6a-8b9c-0d1e2f3a4b02"
	TEST_OWNER_ID         = "5a0c2b7e-1d3f-4e6a-8b9c-0d1e2f3a4b03"
	TEST_ACCOUNT_NUMBER   = "40817643000000000001"
	TEST_CURRENCY_RUB     = "RUB"
	TEST_NOT_UUID         = "abc"
)

// wantGetAccountRequest - запрос в account-service без владельца
func wantGetAccountRequest(accountID uuid.UUID) *accountv1.GetAccountRequest {
	return &accountv1.GetAccountRequest{UserId: "", AccountId: accountID.String()}
}

// liveContext - обычный контекст теста
func liveContext(t *testing.T) context.Context {
	return t.Context()
}

// canceledContext - контекст, отменённый до вызова
func canceledContext(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func TestAccountClient_GetAccount(t *testing.T) {
	accountID := uuid.MustParse(TEST_ACCOUNT_ID)
	ownerID := uuid.MustParse(TEST_OWNER_ID)

	tests := []struct {
		name string
		resp *accountv1.Account
		want *models.AccountInfo
	}{
		{
			name: "active",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Number: TEST_ACCOUNT_NUMBER, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(TEST_CURRENCY_RUB), Status: models.ACCOUNT_STATUS_ACTIVE},
		},
		{
			name: "blocked",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(TEST_CURRENCY_RUB), Status: models.ACCOUNT_STATUS_BLOCKED},
		},
		{
			name: "closed",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(TEST_CURRENCY_RUB), Status: models.ACCOUNT_STATUS_CLOSED},
		},
		{
			name: "unspecified status maps to empty",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(TEST_CURRENCY_RUB), Status: ""},
		},
		{
			name: "unknown status maps to empty",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus(99)},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(TEST_CURRENCY_RUB), Status: ""},
		},
		{
			name: "id taken from request not response",
			resp: &accountv1.Account{Id: TEST_OTHER_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(TEST_CURRENCY_RUB), Status: models.ACCOUNT_STATUS_ACTIVE},
		},
		{
			name: "lowercase currency kept as is",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: "usd", Status: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency("usd"), Status: models.ACCOUNT_STATUS_ACTIVE},
		},
		{
			name: "empty currency kept as is",
			resp: &accountv1.Account{Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Currency: "", Status: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE},
			want: &models.AccountInfo{ID: accountID, UserID: ownerID, Currency: models.Currency(""), Status: models.ACCOUNT_STATUS_ACTIVE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountServiceClient{resp: tt.resp}
			c := NewAccountClient(fake)

			got, err := c.GetAccount(t.Context(), accountID)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			require.Len(t, fake.requests, 1)
			assert.True(t, proto.Equal(wantGetAccountRequest(accountID), fake.requests[0]), "unexpected request: %v", fake.requests[0])
		})
	}
}

func TestAccountClient_GetAccount_NotFound(t *testing.T) {
	accountID := uuid.MustParse(TEST_ACCOUNT_ID)
	fake := &fakeAccountServiceClient{err: status.Error(codes.NotFound, "account not found")}
	c := NewAccountClient(fake)

	got, err := c.GetAccount(t.Context(), accountID)

	require.ErrorIs(t, err, domain.ErrAccountNotFound)
	assert.Nil(t, got)
	require.Len(t, fake.requests, 1)
	assert.True(t, proto.Equal(wantGetAccountRequest(accountID), fake.requests[0]), "unexpected request: %v", fake.requests[0])
}

func TestAccountClient_GetAccount_RPCError(t *testing.T) {
	accountID := uuid.MustParse(TEST_ACCOUNT_ID)

	tests := []struct {
		name     string
		rpcErr   error
		newCtx   func(t *testing.T) context.Context
		wantCode codes.Code
	}{
		{name: "unavailable", rpcErr: status.Error(codes.Unavailable, "connection refused"), newCtx: liveContext, wantCode: codes.Unavailable},
		{name: "internal", rpcErr: status.Error(codes.Internal, "boom"), newCtx: liveContext, wantCode: codes.Internal},
		{name: "canceled context", rpcErr: status.Error(codes.Canceled, "context canceled"), newCtx: canceledContext, wantCode: codes.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountServiceClient{err: tt.rpcErr}
			c := NewAccountClient(fake)
			got, err := c.GetAccount(tt.newCtx(t), accountID)

			require.ErrorIs(t, err, tt.rpcErr)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.NotErrorIs(t, err, domain.ErrAccountNotFound)
			assert.Nil(t, got)
			require.Len(t, fake.requests, 1)
			assert.True(t, proto.Equal(wantGetAccountRequest(accountID), fake.requests[0]), "unexpected request: %v", fake.requests[0])
		})
	}
}

func TestAccountClient_GetAccount_InvalidOwner(t *testing.T) {
	accountID := uuid.MustParse(TEST_ACCOUNT_ID)

	tests := []struct {
		name   string
		userID string
	}{
		{name: "empty user id", userID: ""},
		{name: "non uuid user id", userID: TEST_NOT_UUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountServiceClient{resp: &accountv1.Account{
				Id: TEST_ACCOUNT_ID, UserId: tt.userID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE,
			}}
			c := NewAccountClient(fake)

			got, err := c.GetAccount(t.Context(), accountID)

			require.Error(t, err)
			assert.NotErrorIs(t, err, domain.ErrAccountNotFound)
			assert.Nil(t, got)
		})
	}
}

func TestAccountClient_GetAccountByNumber(t *testing.T) {
	fake := &fakeAccountServiceClient{resp: &accountv1.Account{
		Id: TEST_ACCOUNT_ID, UserId: TEST_OWNER_ID, Number: TEST_ACCOUNT_NUMBER, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED,
	}}
	c := NewAccountClient(fake)

	got, err := c.GetAccountByNumber(t.Context(), TEST_ACCOUNT_NUMBER)

	require.NoError(t, err)
	assert.Equal(t, &models.AccountInfo{
		ID:       uuid.MustParse(TEST_ACCOUNT_ID),
		UserID:   uuid.MustParse(TEST_OWNER_ID),
		Currency: models.Currency(TEST_CURRENCY_RUB),
		Status:   models.ACCOUNT_STATUS_BLOCKED,
	}, got)
	assert.Empty(t, fake.requests)
	require.Len(t, fake.numberRequests, 1)
	assert.True(t, proto.Equal(&accountv1.GetAccountByNumberRequest{Number: TEST_ACCOUNT_NUMBER}, fake.numberRequests[0]), "unexpected request: %v", fake.numberRequests[0])
}

func TestAccountClient_GetAccountByNumber_NotFound(t *testing.T) {
	fake := &fakeAccountServiceClient{err: status.Error(codes.NotFound, "account not found")}
	c := NewAccountClient(fake)

	got, err := c.GetAccountByNumber(t.Context(), TEST_ACCOUNT_NUMBER)

	require.ErrorIs(t, err, domain.ErrAccountNotFound)
	assert.Nil(t, got)
}

func TestAccountClient_GetAccountByNumber_RPCError(t *testing.T) {
	tests := []struct {
		name     string
		rpcErr   error
		wantCode codes.Code
	}{
		{name: "unavailable", rpcErr: status.Error(codes.Unavailable, "connection refused"), wantCode: codes.Unavailable},
		{name: "invalid argument", rpcErr: status.Error(codes.InvalidArgument, "number must be 20 digits"), wantCode: codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountServiceClient{err: tt.rpcErr}
			c := NewAccountClient(fake)

			got, err := c.GetAccountByNumber(t.Context(), TEST_ACCOUNT_NUMBER)

			require.ErrorIs(t, err, tt.rpcErr)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.NotErrorIs(t, err, domain.ErrAccountNotFound)
			assert.Nil(t, got)
		})
	}
}

func TestAccountClient_GetAccountByNumber_InvalidIDs(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		userID    string
	}{
		{name: "empty account id", accountID: "", userID: TEST_OWNER_ID},
		{name: "non uuid account id", accountID: TEST_NOT_UUID, userID: TEST_OWNER_ID},
		{name: "non uuid user id", accountID: TEST_ACCOUNT_ID, userID: TEST_NOT_UUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountServiceClient{resp: &accountv1.Account{
				Id: tt.accountID, UserId: tt.userID, Currency: TEST_CURRENCY_RUB, Status: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE,
			}}
			c := NewAccountClient(fake)

			got, err := c.GetAccountByNumber(t.Context(), TEST_ACCOUNT_NUMBER)

			require.Error(t, err)
			assert.NotErrorIs(t, err, domain.ErrAccountNotFound)
			assert.Nil(t, got)
		})
	}
}
