package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

const (
	TEST_INITIATOR_ID     = "a1b2c3d4-0001-4abc-8def-0000000000a1"
	TEST_OFFICER_ID       = "a1b2c3d4-0002-4abc-8def-0000000000b2"
	TEST_USER_ID          = "a1b2c3d4-0003-4abc-8def-0000000000c3"
	TEST_FROM_ACCOUNT_ID  = "b1b2c3d4-0004-4abc-8def-0000000000d4"
	TEST_TO_ACCOUNT_ID    = "b1b2c3d4-0005-4abc-8def-0000000000e5"
	TEST_ACCOUNT_ID       = "b1b2c3d4-0006-4abc-8def-0000000000f6"
	TEST_TRANSACTION_ID   = "c1b2c3d4-0007-4abc-8def-0000000000a7"
	TEST_TRANSACTION_ID_2 = "c1b2c3d4-0008-4abc-8def-0000000000b8"
	TEST_TRANSACTION_ID_3 = "c1b2c3d4-0009-4abc-8def-0000000000c9"
	TEST_BAD_ID           = "abc"
	TEST_AMOUNT           = 1500
	TEST_BALANCE_AMOUNT   = 12345
	TEST_CURRENCY_RUB     = "RUB"
	TEST_ERROR_LOG        = "level=ERROR"
	TEST_CURRENCY_MESSAGE = "currency is not allowed"
	TEST_ACCOUNT_NUMBER   = "40817810400000000014"
)

// testCreatedAt - фиксированное время создания тестовых транзакций.
func testCreatedAt() time.Time {
	return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
}

// uuidPtr разбирает UUID и возвращает указатель на него.
func uuidPtr(value string) *uuid.UUID {
	parsed := uuid.MustParse(value)
	return &parsed
}

// newTestTransaction - проведённый перевод с двумя проводками.
func newTestTransaction() *models.Transaction {
	return &models.Transaction{
		ID:            uuid.MustParse(TEST_TRANSACTION_ID),
		Type:          models.TYPE_TRANSFER,
		Status:        models.STATUS_COMPLETED,
		FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID),
		ToAccountID:   uuid.MustParse(TEST_TO_ACCOUNT_ID),
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY_RUB,
		InitiatorID:   uuidPtr(TEST_INITIATOR_ID),
		CreatedAt:     testCreatedAt(),
		Entries: []models.Entry{
			newTestEntry(TEST_ENTRY_ID_DEBIT, TEST_FROM_ACCOUNT_ID, models.DIRECTION_DEBIT),
			newTestEntry(TEST_ENTRY_ID_CREDIT, TEST_TO_ACCOUNT_ID, models.DIRECTION_CREDIT),
		},
	}
}

// newDeclinedTransaction - отклонённая транзакция с причиной отказа.
func newDeclinedTransaction(reason string) *models.Transaction {
	tx := newTestTransaction()
	tx.Status = models.STATUS_DECLINED
	tx.DeclineReason = reason
	tx.Entries = nil
	return tx
}

// rpcCase - один RPC с валидным запросом и его фиксированной Internal-ошибкой.
type rpcCase struct {
	name        string
	call        func(ctx context.Context, s *LedgerServer) (proto.Message, error)
	internalErr error
}

// allRPCs - все пять RPC с валидными запросами.
func allRPCs() []rpcCase {
	return []rpcCase{
		{
			name: "Transfer",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Transfer(ctx, &ledgerv1.TransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB})
			},
			internalErr: domain.ErrTransferFailed,
		},
		{
			name: "Deposit",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Deposit(ctx, &ledgerv1.DepositRequest{AccountId: TEST_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, OfficerId: TEST_OFFICER_ID})
			},
			internalErr: domain.ErrDepositFailed,
		},
		{
			name: "GetTransaction",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.GetTransaction(ctx, &ledgerv1.GetTransactionRequest{TransactionId: TEST_TRANSACTION_ID})
			},
			internalErr: domain.ErrGetFailed,
		},
		{
			name: "ListTransactions",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.ListTransactions(ctx, &ledgerv1.ListTransactionsRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID, Limit: 20, Offset: 0})
			},
			internalErr: domain.ErrListFailed,
		},
		{
			name: "GetBalance",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.GetBalance(ctx, &ledgerv1.GetBalanceRequest{AccountId: TEST_ACCOUNT_ID})
			},
			internalErr: domain.ErrBalanceFailed,
		},
	}
}

func TestLedgerServer_Transfer(t *testing.T) {
	validRequest := func() *ledgerv1.TransferRequest {
		return &ledgerv1.TransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: "rub"}
	}
	validCommand := models.TransferCommand{
		InitiatorID: uuid.MustParse(TEST_INITIATOR_ID),
		From:        uuid.MustParse(TEST_FROM_ACCOUNT_ID),
		To:          uuid.MustParse(TEST_TO_ACCOUNT_ID),
		Amount:      TEST_AMOUNT,
		Currency:    "rub",
	}
	tests := []struct {
		name      string
		modify    func(in *ledgerv1.TransferRequest)
		wantCalls []models.TransferCommand
		wantErr   error
	}{
		{
			name:      "valid request passed as is",
			modify:    func(in *ledgerv1.TransferRequest) {},
			wantCalls: []models.TransferCommand{validCommand},
		},
		{
			name: "zero amount and empty currency passed as is",
			modify: func(in *ledgerv1.TransferRequest) {
				in.Amount = 0
				in.Currency = ""
			},
			wantCalls: []models.TransferCommand{{InitiatorID: validCommand.InitiatorID, From: validCommand.From, To: validCommand.To, Amount: 0, Currency: ""}},
		},
		{
			name:      "negative amount passed as is",
			modify:    func(in *ledgerv1.TransferRequest) { in.Amount = -1 },
			wantCalls: []models.TransferCommand{{InitiatorID: validCommand.InitiatorID, From: validCommand.From, To: validCommand.To, Amount: -1, Currency: "rub"}},
		},
		{
			name: "upper case ids",
			modify: func(in *ledgerv1.TransferRequest) {
				in.InitiatorId = strings.ToUpper(TEST_INITIATOR_ID)
				in.FromAccountId = strings.ToUpper(TEST_FROM_ACCOUNT_ID)
				in.ToAccountId = strings.ToUpper(TEST_TO_ACCOUNT_ID)
			},
			wantCalls: []models.TransferCommand{validCommand},
		},
		{
			name:    "initiator_id not uuid",
			modify:  func(in *ledgerv1.TransferRequest) { in.InitiatorId = TEST_BAD_ID },
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:    "initiator_id empty",
			modify:  func(in *ledgerv1.TransferRequest) { in.InitiatorId = "" },
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:    "from_account_id not uuid",
			modify:  func(in *ledgerv1.TransferRequest) { in.FromAccountId = TEST_BAD_ID },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "from_account_id empty",
			modify:  func(in *ledgerv1.TransferRequest) { in.FromAccountId = "" },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "to_account_id not uuid",
			modify:  func(in *ledgerv1.TransferRequest) { in.ToAccountId = TEST_BAD_ID },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "to_account_id empty",
			modify:  func(in *ledgerv1.TransferRequest) { in.ToAccountId = "" },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "initiator_id checked before from_account_id",
			modify: func(in *ledgerv1.TransferRequest) {
				in.InitiatorId = TEST_BAD_ID
				in.FromAccountId = TEST_BAD_ID
			},
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name: "from empty and to not uuid",
			modify: func(in *ledgerv1.TransferRequest) {
				in.FromAccountId = ""
				in.ToAccountId = TEST_BAD_ID
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			fake := &fakeLedgerService{transferTx: tx}
			srv, _ := newTestServer(fake)
			in := validRequest()
			tt.modify(in)

			got, err := srv.Transfer(t.Context(), in)

			assert.Equal(t, tt.wantCalls, fake.transferCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			want := transactionToProto(tx)
			assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
		})
	}
}

func TestLedgerServer_Deposit(t *testing.T) {
	validRequest := func() *ledgerv1.DepositRequest {
		return &ledgerv1.DepositRequest{AccountId: TEST_ACCOUNT_ID, Amount: 5000, Currency: TEST_CURRENCY_RUB, OfficerId: TEST_OFFICER_ID}
	}
	validCommand := models.DepositCommand{
		AccountID: uuid.MustParse(TEST_ACCOUNT_ID),
		Amount:    5000,
		Currency:  TEST_CURRENCY_RUB,
		OfficerID: uuid.MustParse(TEST_OFFICER_ID),
	}
	tests := []struct {
		name      string
		modify    func(in *ledgerv1.DepositRequest)
		wantCalls []models.DepositCommand
		wantErr   error
	}{
		{
			name:      "valid request passed as is",
			modify:    func(in *ledgerv1.DepositRequest) {},
			wantCalls: []models.DepositCommand{validCommand},
		},
		{
			name: "zero amount and lower case currency passed as is",
			modify: func(in *ledgerv1.DepositRequest) {
				in.Amount = 0
				in.Currency = "rub"
			},
			wantCalls: []models.DepositCommand{{AccountID: validCommand.AccountID, Amount: 0, Currency: "rub", OfficerID: validCommand.OfficerID}},
		},
		{
			name: "upper case ids",
			modify: func(in *ledgerv1.DepositRequest) {
				in.AccountId = strings.ToUpper(TEST_ACCOUNT_ID)
				in.OfficerId = strings.ToUpper(TEST_OFFICER_ID)
			},
			wantCalls: []models.DepositCommand{validCommand},
		},
		{
			name:    "account_id not uuid",
			modify:  func(in *ledgerv1.DepositRequest) { in.AccountId = TEST_BAD_ID },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "account_id empty",
			modify:  func(in *ledgerv1.DepositRequest) { in.AccountId = "" },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "officer_id not uuid",
			modify:  func(in *ledgerv1.DepositRequest) { in.OfficerId = TEST_BAD_ID },
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name:    "officer_id empty",
			modify:  func(in *ledgerv1.DepositRequest) { in.OfficerId = "" },
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name: "account_id checked before officer_id",
			modify: func(in *ledgerv1.DepositRequest) {
				in.AccountId = TEST_BAD_ID
				in.OfficerId = TEST_BAD_ID
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			tx.Type = models.TYPE_DEPOSIT
			fake := &fakeLedgerService{depositTx: tx}
			srv, _ := newTestServer(fake)
			in := validRequest()
			tt.modify(in)

			got, err := srv.Deposit(t.Context(), in)

			assert.Equal(t, tt.wantCalls, fake.depositCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			want := transactionToProto(tx)
			assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
		})
	}
}

func TestLedgerServer_GetTransaction(t *testing.T) {
	tests := []struct {
		name          string
		transactionID string
		wantCalls     []uuid.UUID
		wantErr       error
	}{
		{name: "valid id", transactionID: TEST_TRANSACTION_ID, wantCalls: []uuid.UUID{uuid.MustParse(TEST_TRANSACTION_ID)}},
		{name: "upper case id", transactionID: strings.ToUpper(TEST_TRANSACTION_ID), wantCalls: []uuid.UUID{uuid.MustParse(TEST_TRANSACTION_ID)}},
		{name: "not uuid", transactionID: TEST_BAD_ID, wantErr: domain.ErrTransactionIDMustBeUUID},
		{name: "empty", transactionID: "", wantErr: domain.ErrTransactionIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			fake := &fakeLedgerService{getTx: tx}
			srv, _ := newTestServer(fake)

			got, err := srv.GetTransaction(t.Context(), &ledgerv1.GetTransactionRequest{TransactionId: tt.transactionID})

			assert.Equal(t, tt.wantCalls, fake.getCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			want := transactionToProto(tx)
			assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
		})
	}
}

func TestLedgerServer_ListTransactions(t *testing.T) {
	second := newTestTransaction()
	second.ID = uuid.MustParse(TEST_TRANSACTION_ID_2)
	second.Type = models.TYPE_DEPOSIT
	twoTransactions := []*models.Transaction{newTestTransaction(), second}
	accountID := uuid.MustParse(TEST_ACCOUNT_ID)

	tests := []struct {
		name      string
		in        *ledgerv1.ListTransactionsRequest
		txs       []*models.Transaction
		wantCalls []listCall
		wantErr   error
	}{
		{
			name:      "valid user and page",
			in:        &ledgerv1.ListTransactionsRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID, Limit: 10, Offset: 5},
			txs:       twoTransactions,
			wantCalls: []listCall{{userID: uuidPtr(TEST_USER_ID), accountID: accountID, page: models.Page{Limit: 10, Offset: 5}}},
		},
		{
			name:      "empty user and zero page",
			in:        &ledgerv1.ListTransactionsRequest{UserId: "", AccountId: TEST_ACCOUNT_ID, Limit: 0, Offset: 0},
			txs:       twoTransactions,
			wantCalls: []listCall{{userID: nil, accountID: accountID, page: models.Page{Limit: 0, Offset: 0}}},
		},
		{
			name:      "large page passed as is",
			in:        &ledgerv1.ListTransactionsRequest{AccountId: TEST_ACCOUNT_ID, Limit: 1000, Offset: math.MaxInt32},
			txs:       twoTransactions,
			wantCalls: []listCall{{userID: nil, accountID: accountID, page: models.Page{Limit: 1000, Offset: math.MaxInt32}}},
		},
		{
			name:      "negative page passed as is",
			in:        &ledgerv1.ListTransactionsRequest{AccountId: TEST_ACCOUNT_ID, Limit: -1, Offset: -5},
			txs:       twoTransactions,
			wantCalls: []listCall{{userID: nil, accountID: accountID, page: models.Page{Limit: -1, Offset: -5}}},
		},
		{
			name:      "upper case ids",
			in:        &ledgerv1.ListTransactionsRequest{UserId: strings.ToUpper(TEST_USER_ID), AccountId: strings.ToUpper(TEST_ACCOUNT_ID), Limit: 20},
			txs:       twoTransactions,
			wantCalls: []listCall{{userID: uuidPtr(TEST_USER_ID), accountID: accountID, page: models.Page{Limit: 20, Offset: 0}}},
		},
		{
			name:      "service returns empty slice",
			in:        &ledgerv1.ListTransactionsRequest{AccountId: TEST_ACCOUNT_ID},
			txs:       []*models.Transaction{},
			wantCalls: []listCall{{userID: nil, accountID: accountID, page: models.Page{}}},
		},
		{
			name:      "service returns nil slice",
			in:        &ledgerv1.ListTransactionsRequest{AccountId: TEST_ACCOUNT_ID},
			txs:       nil,
			wantCalls: []listCall{{userID: nil, accountID: accountID, page: models.Page{}}},
		},
		{
			name:    "user_id not uuid",
			in:      &ledgerv1.ListTransactionsRequest{UserId: TEST_BAD_ID, AccountId: TEST_ACCOUNT_ID},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "account_id not uuid",
			in:      &ledgerv1.ListTransactionsRequest{UserId: TEST_USER_ID, AccountId: TEST_BAD_ID},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "account_id empty with empty user",
			in:      &ledgerv1.ListTransactionsRequest{UserId: "", AccountId: ""},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "user_id checked before account_id",
			in:      &ledgerv1.ListTransactionsRequest{UserId: TEST_BAD_ID, AccountId: TEST_BAD_ID},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLedgerService{listTxs: tt.txs}
			srv, _ := newTestServer(fake)

			got, err := srv.ListTransactions(t.Context(), tt.in)

			assert.Equal(t, tt.wantCalls, fake.listCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			want := &ledgerv1.ListTransactionsResponse{Transactions: transactionsToProto(tt.txs)}
			assert.Len(t, got.GetTransactions(), len(tt.txs))
			assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
		})
	}
}

func TestLedgerServer_GetBalance(t *testing.T) {
	accountID := uuid.MustParse(TEST_ACCOUNT_ID)
	tests := []struct {
		name      string
		accountID string
		balance   *models.Balance
		wantCalls []uuid.UUID
		want      *ledgerv1.GetBalanceResponse
		wantErr   error
	}{
		{
			name:      "valid account",
			accountID: TEST_ACCOUNT_ID,
			balance:   &models.Balance{AccountID: accountID, Amount: TEST_BALANCE_AMOUNT, Currency: TEST_CURRENCY_RUB},
			wantCalls: []uuid.UUID{accountID},
			want:      &ledgerv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: TEST_BALANCE_AMOUNT, Currency: TEST_CURRENCY_RUB},
		},
		{
			name:      "zero amount",
			accountID: TEST_ACCOUNT_ID,
			balance:   &models.Balance{AccountID: accountID, Amount: 0, Currency: "USD"},
			wantCalls: []uuid.UUID{accountID},
			want:      &ledgerv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: 0, Currency: "USD"},
		},
		{
			name:      "upper case id",
			accountID: strings.ToUpper(TEST_ACCOUNT_ID),
			balance:   &models.Balance{AccountID: accountID, Amount: TEST_BALANCE_AMOUNT, Currency: TEST_CURRENCY_RUB},
			wantCalls: []uuid.UUID{accountID},
			want:      &ledgerv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: TEST_BALANCE_AMOUNT, Currency: TEST_CURRENCY_RUB},
		},
		{name: "not uuid", accountID: TEST_BAD_ID, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "empty", accountID: "", wantErr: domain.ErrAccountIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLedgerService{balance: tt.balance}
			srv, _ := newTestServer(fake)

			got, err := srv.GetBalance(t.Context(), &ledgerv1.GetBalanceRequest{AccountId: tt.accountID})

			assert.Equal(t, tt.wantCalls, fake.balanceCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Truef(t, proto.Equal(tt.want, got), "want %v, got %v", tt.want, got)
		})
	}
}

func TestLedgerServer_ParseErrorDoesNotCallService(t *testing.T) {
	tests := []struct {
		name    string
		call    func(ctx context.Context, s *LedgerServer) (proto.Message, error)
		wantErr error
	}{
		{
			name: "Transfer initiator_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Transfer(ctx, &ledgerv1.TransferRequest{InitiatorId: TEST_BAD_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB})
			},
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name: "Transfer from_account_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Transfer(ctx, &ledgerv1.TransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: "", ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB})
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "Transfer to_account_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Transfer(ctx, &ledgerv1.TransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: TEST_BAD_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB})
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "Deposit account_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Deposit(ctx, &ledgerv1.DepositRequest{AccountId: TEST_BAD_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, OfficerId: TEST_OFFICER_ID})
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "Deposit officer_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.Deposit(ctx, &ledgerv1.DepositRequest{AccountId: TEST_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB, OfficerId: ""})
			},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name: "GetTransaction transaction_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.GetTransaction(ctx, &ledgerv1.GetTransactionRequest{TransactionId: TEST_BAD_ID})
			},
			wantErr: domain.ErrTransactionIDMustBeUUID,
		},
		{
			name: "ListTransactions user_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.ListTransactions(ctx, &ledgerv1.ListTransactionsRequest{UserId: TEST_BAD_ID, AccountId: TEST_ACCOUNT_ID})
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "ListTransactions account_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.ListTransactions(ctx, &ledgerv1.ListTransactionsRequest{UserId: TEST_USER_ID, AccountId: ""})
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "GetBalance account_id",
			call: func(ctx context.Context, s *LedgerServer) (proto.Message, error) {
				return s.GetBalance(ctx, &ledgerv1.GetBalanceRequest{AccountId: TEST_BAD_ID})
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// фейк в режиме успеха: если сервер его вызовет, ошибки разбора не будет
			fake := &fakeLedgerService{
				transferTx: newTestTransaction(),
				depositTx:  newTestTransaction(),
				getTx:      newTestTransaction(),
				listTxs:    []*models.Transaction{newTestTransaction()},
				balance:    &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: TEST_BALANCE_AMOUNT, Currency: TEST_CURRENCY_RUB},
			}
			srv, _ := newTestServer(fake)

			got, err := tt.call(t.Context(), srv)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
			assert.Nil(t, got)
			assert.Equal(t, 0, fake.totalCalls())
		})
	}
}

func TestLedgerServer_DeclinedWithTransaction(t *testing.T) {
	rpcs := allRPCs()[:2] // только Transfer и Deposit возвращают отклонённую транзакцию
	tests := []struct {
		name   string
		reason string
		err    error
	}{
		{name: "insufficient funds", reason: TEST_DECLINE_REASON, err: domain.ErrInsufficientFunds},
		{name: "wrapped insufficient funds", reason: TEST_DECLINE_REASON, err: fmt.Errorf("transfer: %w", domain.ErrInsufficientFunds)},
		{name: "transfer denied", reason: "denied by antifraud (check chk-1)", err: domain.ErrTransferDenied},
		{name: "wrapped transfer denied", reason: "denied by antifraud (check chk-1)", err: fmt.Errorf("transfer: %w", domain.ErrTransferDenied)},
		{name: "empty decline reason", reason: "", err: domain.ErrInsufficientFunds},
	}
	for _, rpc := range rpcs {
		for _, tt := range tests {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				tx := newDeclinedTransaction(tt.reason)
				fake := newFailingLedgerService(tt.err)
				fake.transferTx = tx
				fake.depositTx = tx
				srv, _ := newTestServer(fake)
				wantErr := status.Error(codes.FailedPrecondition, fmt.Sprintf("transaction %s declined: %s", TEST_TRANSACTION_ID, tt.reason))

				got, err := rpc.call(t.Context(), srv)

				require.ErrorIs(t, err, wantErr)
				assert.Equal(t, codes.FailedPrecondition, status.Code(err))
				assert.Equal(t, status.Convert(wantErr).Message(), status.Convert(err).Message())
				assert.Nil(t, got)
			})
		}
	}
}

func TestLedgerServer_DeclinedWithoutTransaction(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "insufficient funds", err: domain.ErrInsufficientFunds},
		{name: "wrapped transfer denied", err: fmt.Errorf("transfer: %w", domain.ErrTransferDenied)},
	}
	for _, rpc := range allRPCs() {
		for _, tt := range tests {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				fake := newFailingLedgerService(tt.err)
				srv, _ := newTestServer(fake)

				got, err := rpc.call(t.Context(), srv)

				assert.Equal(t, codes.FailedPrecondition, status.Code(err))
				assert.Equal(t, tt.err.Error(), status.Convert(err).Message())
				assert.Nil(t, got)
			})
		}
	}
}

func TestLedgerServer_DomainErrorMapping(t *testing.T) {
	currencyNotAllowed := status.Error(codes.InvalidArgument, TEST_CURRENCY_MESSAGE)
	tests := []struct {
		name    string
		err     error
		wantErr error
		wantLog bool
	}{
		{name: "transaction not found", err: domain.ErrTransactionNotFound, wantErr: domain.ErrTransactionMissing},
		{name: "wrapped transaction not found", err: fmt.Errorf("get: %w", domain.ErrTransactionNotFound), wantErr: domain.ErrTransactionMissing},
		{name: "balance not found", err: domain.ErrBalanceNotFound, wantErr: domain.ErrBalanceMissing},
		{name: "wrapped balance not found", err: fmt.Errorf("balance: %w", domain.ErrBalanceNotFound), wantErr: domain.ErrBalanceMissing},
		{name: "account not found", err: domain.ErrAccountNotFound, wantErr: domain.ErrAccountMissing},
		{name: "wrapped account not found", err: fmt.Errorf("get account: %w", domain.ErrAccountNotFound), wantErr: domain.ErrAccountMissing},
		{name: "account not owned", err: domain.ErrAccountNotOwned, wantErr: domain.ErrForeignAccount},
		{name: "wrapped account not owned", err: fmt.Errorf("check owner: %w", domain.ErrAccountNotOwned), wantErr: domain.ErrForeignAccount},
		{name: "account not active", err: domain.ErrAccountNotActive, wantErr: domain.ErrAccountIsNotActive},
		{name: "wrapped account not active", err: fmt.Errorf("check status: %w", domain.ErrAccountNotActive), wantErr: domain.ErrAccountIsNotActive},
		{name: "currency mismatch", err: domain.ErrCurrencyMismatch, wantErr: domain.ErrCurrencyIsMismatch},
		{name: "wrapped currency mismatch", err: fmt.Errorf("check currency: %w", domain.ErrCurrencyMismatch), wantErr: domain.ErrCurrencyIsMismatch},
		{name: "same account", err: domain.ErrSameAccount, wantErr: domain.ErrSameAccountTransfer},
		{name: "wrapped same account", err: fmt.Errorf("transfer: %w", domain.ErrSameAccount), wantErr: domain.ErrSameAccountTransfer},
		{name: "currency not allowed", err: domain.ErrCurrencyNotAllowed, wantErr: currencyNotAllowed},
		{name: "fraud check failed", err: domain.ErrFraudCheckFailed, wantErr: domain.ErrAntifraudIsDown, wantLog: true},
		{name: "wrapped fraud check failed", err: fmt.Errorf("check transfer: %w", domain.ErrFraudCheckFailed), wantErr: domain.ErrAntifraudIsDown, wantLog: true},
	}
	for _, rpc := range allRPCs() {
		for _, tt := range tests {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				fake := newFailingLedgerService(tt.err)
				srv, logBuf := newTestServer(fake)

				got, err := rpc.call(t.Context(), srv)

				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, status.Code(tt.wantErr), status.Code(err))
				assert.Nil(t, got)
				assert.Equal(t, tt.wantLog, strings.Contains(logBuf.String(), TEST_ERROR_LOG))
			})
		}
	}
}

func TestLedgerServer_InternalErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "plain error", err: errors.New("db failure")},
		{name: "wrapped plain error", err: fmt.Errorf("deposit: no cash account for RUB")},
		{name: "context canceled", err: context.Canceled},
		{name: "wrapped context canceled", err: fmt.Errorf("query: %w", context.Canceled)},
	}
	for _, rpc := range allRPCs() {
		for _, tt := range tests {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				fake := newFailingLedgerService(tt.err)
				srv, logBuf := newTestServer(fake)

				got, err := rpc.call(t.Context(), srv)

				require.ErrorIs(t, err, rpc.internalErr)
				assert.Equal(t, codes.Internal, status.Code(err))
				assert.Nil(t, got)
				assert.Contains(t, logBuf.String(), TEST_ERROR_LOG)
			})
		}
	}
}

func TestLedgerServer_NoUnknownCode(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{name: "plain error", err: errors.New("db failure")},
		{name: "context canceled", err: context.Canceled},
		{name: "sentinel", err: domain.ErrAccountNotFound},
		{name: "wrapped sentinel", err: fmt.Errorf("check: %w", domain.ErrCurrencyMismatch)},
		{name: "fraud check failed", err: domain.ErrFraudCheckFailed},
		{name: "insufficient funds without transaction", err: domain.ErrInsufficientFunds},
		{name: "upstream status", err: status.Error(codes.Unavailable, "connection refused")},
		{name: "wrapped upstream status", err: fmt.Errorf("get account: %w", status.Error(codes.Unavailable, "connection refused"))},
	}
	for _, rpc := range allRPCs() {
		for _, tt := range errs {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				fake := newFailingLedgerService(tt.err)
				srv, _ := newTestServer(fake)

				got, err := rpc.call(t.Context(), srv)

				assert.NotEqual(t, codes.OK, status.Code(err))
				assert.NotEqual(t, codes.Unknown, status.Code(err))
				assert.Nil(t, got)
			})
		}
	}
}

func TestLedgerServer_TransferByNumber(t *testing.T) {
	validRequest := func() *ledgerv1.TransferRequest {
		return &ledgerv1.TransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountNumber: TEST_ACCOUNT_NUMBER, Amount: TEST_AMOUNT, Currency: "rub"}
	}
	tests := []struct {
		name      string
		modify    func(in *ledgerv1.TransferRequest)
		wantCalls []models.TransferCommand
		wantErr   error
	}{
		{
			// id получателя сервис узнаёт сам, To остаётся пустым
			name:   "number passed as is",
			modify: func(in *ledgerv1.TransferRequest) {},
			wantCalls: []models.TransferCommand{{
				InitiatorID: uuid.MustParse(TEST_INITIATOR_ID),
				From:        uuid.MustParse(TEST_FROM_ACCOUNT_ID),
				Amount:      TEST_AMOUNT,
				Currency:    "rub",
				ToNumber:    TEST_ACCOUNT_NUMBER,
			}},
		},
		{name: "19 digits", modify: func(in *ledgerv1.TransferRequest) { in.ToAccountNumber = TEST_ACCOUNT_NUMBER[1:] }, wantErr: domain.ErrAccountNumberIsInvalid},
		{name: "21 digits", modify: func(in *ledgerv1.TransferRequest) { in.ToAccountNumber = TEST_ACCOUNT_NUMBER + "0" }, wantErr: domain.ErrAccountNumberIsInvalid},
		{name: "letters", modify: func(in *ledgerv1.TransferRequest) { in.ToAccountNumber = "4081781040000000001A" }, wantErr: domain.ErrAccountNumberIsInvalid},
		{name: "spaces", modify: func(in *ledgerv1.TransferRequest) { in.ToAccountNumber = " 4081781040000000001" }, wantErr: domain.ErrAccountNumberIsInvalid},
		{
			name:    "initiator checked before number",
			modify:  func(in *ledgerv1.TransferRequest) { in.InitiatorId = TEST_BAD_ID; in.ToAccountNumber = "x" },
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:    "from checked before number",
			modify:  func(in *ledgerv1.TransferRequest) { in.FromAccountId = TEST_BAD_ID; in.ToAccountNumber = "x" },
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			fake := &fakeLedgerService{transferTx: tx}
			srv, _ := newTestServer(fake)
			in := validRequest()
			tt.modify(in)

			got, err := srv.Transfer(t.Context(), in)

			assert.Equal(t, tt.wantCalls, fake.transferCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			want := transactionToProto(tx)
			assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
		})
	}
}

func TestLedgerServer_TransferByNumber_RecipientNotFound(t *testing.T) {
	fake := newFailingLedgerService(fmt.Errorf("number %s: %w", TEST_ACCOUNT_NUMBER, domain.ErrAccountNotFound))
	srv, logBuf := newTestServer(fake)

	got, err := srv.Transfer(t.Context(), &ledgerv1.TransferRequest{
		InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountNumber: TEST_ACCOUNT_NUMBER, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB,
	})

	require.ErrorIs(t, err, domain.ErrAccountMissing)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, "account not found", status.Convert(err).Message())
	assert.Nil(t, got)
	assert.NotContains(t, logBuf.String(), TEST_ERROR_LOG)
}
