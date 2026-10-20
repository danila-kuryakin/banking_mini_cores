package client

import (
	"errors"
	"math"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/ledger/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TEST_ACCOUNT_ID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	TEST_RUB        = models.Currency("RUB")
	TEST_USD        = models.Currency("USD")
)

func TestLedgerClient_GetBalance(t *testing.T) {
	tests := []struct {
		name          string
		accountID     uuid.UUID
		currency      models.Currency
		amount        int64
		want          *models.Balance
		wantAccountID string
	}{
		{name: "rub positive", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_RUB, amount: 15000, want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 15000, Currency: TEST_RUB}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "zero amount", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_RUB, amount: 0, want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_RUB}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "negative amount", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_RUB, amount: -1, want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: -1, Currency: TEST_RUB}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "max int64", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_RUB, amount: math.MaxInt64, want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: math.MaxInt64, Currency: TEST_RUB}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "min int64", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_RUB, amount: math.MinInt64, want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: math.MinInt64, Currency: TEST_RUB}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "usd", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_USD, amount: 250, want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 250, Currency: TEST_USD}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "nil account id", accountID: uuid.Nil, currency: TEST_RUB, amount: 100, want: &models.Balance{AccountID: uuid.Nil, Amount: 100, Currency: TEST_RUB}, wantAccountID: TEST_NIL_UUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLedgerServiceClient{resp: &ledgerv1.GetBalanceResponse{
				AccountId: tt.accountID.String(),
				Amount:    tt.amount,
				Currency:  string(tt.currency),
			}}
			ledgerClient := NewLedgerClient(fake)

			got, err := ledgerClient.GetBalance(t.Context(), tt.accountID, tt.currency)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, []string{tt.wantAccountID}, fake.accountIDs)
		})
	}
}

func TestLedgerClient_GetBalance_NotFoundIsZero(t *testing.T) {
	tests := []struct {
		name          string
		accountID     uuid.UUID
		currency      models.Currency
		message       string
		want          *models.Balance
		wantAccountID string
	}{
		{name: "usd", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_USD, message: "balance not found", want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_USD}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "rub", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_RUB, message: "balance not found", want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_RUB}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "empty message", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_USD, message: "", want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_USD}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "arbitrary message", accountID: uuid.MustParse(TEST_ACCOUNT_ID), currency: TEST_USD, message: "whatever", want: &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_USD}, wantAccountID: TEST_ACCOUNT_ID},
		{name: "nil account id", accountID: uuid.Nil, currency: TEST_RUB, message: "balance not found", want: &models.Balance{AccountID: uuid.Nil, Amount: 0, Currency: TEST_RUB}, wantAccountID: TEST_NIL_UUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLedgerServiceClient{getBalanceErr: status.Error(codes.NotFound, tt.message)}
			ledgerClient := NewLedgerClient(fake)

			got, err := ledgerClient.GetBalance(t.Context(), tt.accountID, tt.currency)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, []string{tt.wantAccountID}, fake.accountIDs)
		})
	}
}

func TestLedgerClient_GetBalance_UpstreamError(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		wantCode codes.Code
	}{
		{name: "unavailable", upstream: status.Error(codes.Unavailable, "ledger-service down"), wantCode: codes.Unavailable},
		{name: "internal", upstream: status.Error(codes.Internal, "internal"), wantCode: codes.Internal},
		{name: "deadline exceeded", upstream: status.Error(codes.DeadlineExceeded, "deadline"), wantCode: codes.DeadlineExceeded},
		{name: "invalid argument", upstream: status.Error(codes.InvalidArgument, "bad account id"), wantCode: codes.InvalidArgument},
		// обычная ошибка без статуса даёт codes.Unknown
		{name: "plain error", upstream: errors.New("boom"), wantCode: codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLedgerServiceClient{getBalanceErr: tt.upstream}
			ledgerClient := NewLedgerClient(fake)

			got, err := ledgerClient.GetBalance(t.Context(), uuid.MustParse(TEST_ACCOUNT_ID), TEST_RUB)

			require.ErrorIs(t, err, tt.upstream)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Nil(t, got)
			assert.Equal(t, []string{TEST_ACCOUNT_ID}, fake.accountIDs)
		})
	}
}
