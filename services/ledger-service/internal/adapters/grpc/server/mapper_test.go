package server

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

const (
	TEST_ENTRY_ID_DEBIT  = "d1b2c3d4-0011-4abc-8def-0000000000a1"
	TEST_ENTRY_ID_CREDIT = "d1b2c3d4-0012-4abc-8def-0000000000b2"
	TEST_ENTRY_ID_THIRD  = "d1b2c3d4-0013-4abc-8def-0000000000c3"
	TEST_DECLINE_REASON  = "insufficient funds"
	TEST_NIL_UUID        = "00000000-0000-0000-0000-000000000000"
)

func TestTransactionToProto_FullTransfer(t *testing.T) {
	tx := newTestTransaction()
	// владельцы счетов заданы, но в proto не попадают
	tx.FromUserID = uuidPtr(TEST_USER_ID)
	tx.ToUserID = uuidPtr(TEST_OFFICER_ID)
	want := &ledgerv1.Transaction{
		Id:            TEST_TRANSACTION_ID,
		Type:          ledgerv1.TransactionType_TRANSACTION_TYPE_TRANSFER,
		Status:        ledgerv1.TransactionStatus_TRANSACTION_STATUS_COMPLETED,
		FromAccountId: TEST_FROM_ACCOUNT_ID,
		ToAccountId:   TEST_TO_ACCOUNT_ID,
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY_RUB,
		InitiatorId:   TEST_INITIATOR_ID,
		DeclineReason: "",
		CreatedAt:     timestamppb.New(testCreatedAt()),
		Entries: []*ledgerv1.Entry{
			{Id: TEST_ENTRY_ID_DEBIT, AccountId: TEST_FROM_ACCOUNT_ID, Direction: ledgerv1.EntryDirection_ENTRY_DIRECTION_DEBIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB},
			{Id: TEST_ENTRY_ID_CREDIT, AccountId: TEST_TO_ACCOUNT_ID, Direction: ledgerv1.EntryDirection_ENTRY_DIRECTION_CREDIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB},
		},
	}

	got := transactionToProto(tx)

	require.NotNil(t, got)
	assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
	assert.True(t, testCreatedAt().Equal(got.GetCreatedAt().AsTime()))
}

func TestTransactionToProto_DeclinedDepositWithoutInitiator(t *testing.T) {
	tx := &models.Transaction{
		ID:            uuid.MustParse(TEST_TRANSACTION_ID),
		Type:          models.TYPE_DEPOSIT,
		Status:        models.STATUS_DECLINED,
		FromAccountID: uuid.Nil,
		ToAccountID:   uuid.MustParse(TEST_ACCOUNT_ID),
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY_RUB,
		InitiatorID:   nil,
		DeclineReason: TEST_DECLINE_REASON,
		CreatedAt:     testCreatedAt(),
	}
	want := &ledgerv1.Transaction{
		Id:            TEST_TRANSACTION_ID,
		Type:          ledgerv1.TransactionType_TRANSACTION_TYPE_DEPOSIT,
		Status:        ledgerv1.TransactionStatus_TRANSACTION_STATUS_DECLINED,
		FromAccountId: TEST_NIL_UUID,
		ToAccountId:   TEST_ACCOUNT_ID,
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY_RUB,
		InitiatorId:   "",
		DeclineReason: TEST_DECLINE_REASON,
		CreatedAt:     timestamppb.New(testCreatedAt()),
	}

	got := transactionToProto(tx)

	require.NotNil(t, got)
	assert.Truef(t, proto.Equal(want, got), "want %v, got %v", want, got)
	assert.Len(t, got.GetEntries(), 0)
}

func TestTransactionToProto_Type(t *testing.T) {
	tests := []struct {
		name   string
		txType models.TransactionType
		want   ledgerv1.TransactionType
	}{
		{name: "transfer", txType: models.TYPE_TRANSFER, want: ledgerv1.TransactionType_TRANSACTION_TYPE_TRANSFER},
		{name: "deposit", txType: models.TYPE_DEPOSIT, want: ledgerv1.TransactionType_TRANSACTION_TYPE_DEPOSIT},
		{name: "empty", txType: "", want: ledgerv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED},
		{name: "unknown", txType: "refund", want: ledgerv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED},
		{name: "upper case", txType: "TRANSFER", want: ledgerv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			tx.Type = tt.txType

			got := transactionToProto(tx)

			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.GetType())
		})
	}
}

func TestTransactionToProto_Status(t *testing.T) {
	tests := []struct {
		name     string
		txStatus models.TransactionStatus
		want     ledgerv1.TransactionStatus
	}{
		{name: "completed", txStatus: models.STATUS_COMPLETED, want: ledgerv1.TransactionStatus_TRANSACTION_STATUS_COMPLETED},
		{name: "declined", txStatus: models.STATUS_DECLINED, want: ledgerv1.TransactionStatus_TRANSACTION_STATUS_DECLINED},
		{name: "empty", txStatus: "", want: ledgerv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED},
		{name: "unknown", txStatus: "pending", want: ledgerv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED},
		{name: "upper case", txStatus: "COMPLETED", want: ledgerv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			tx.Status = tt.txStatus

			got := transactionToProto(tx)

			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.GetStatus())
		})
	}
}

func TestTransactionToProto_EntryDirection(t *testing.T) {
	tests := []struct {
		name      string
		direction models.Direction
		want      ledgerv1.EntryDirection
	}{
		{name: "debit", direction: models.DIRECTION_DEBIT, want: ledgerv1.EntryDirection_ENTRY_DIRECTION_DEBIT},
		{name: "credit", direction: models.DIRECTION_CREDIT, want: ledgerv1.EntryDirection_ENTRY_DIRECTION_CREDIT},
		{name: "empty", direction: "", want: ledgerv1.EntryDirection_ENTRY_DIRECTION_UNSPECIFIED},
		{name: "unknown", direction: "both", want: ledgerv1.EntryDirection_ENTRY_DIRECTION_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			tx.Entries = []models.Entry{newTestEntry(TEST_ENTRY_ID_DEBIT, TEST_FROM_ACCOUNT_ID, tt.direction)}

			got := transactionToProto(tx)

			require.NotNil(t, got)
			require.Len(t, got.GetEntries(), 1)
			assert.Equal(t, tt.want, got.GetEntries()[0].GetDirection())
		})
	}
}

func TestTransactionToProto_NoEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []models.Entry
	}{
		{name: "nil entries", entries: nil},
		{name: "empty entries", entries: []models.Entry{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			tx.Entries = tt.entries

			got := transactionToProto(tx)

			require.NotNil(t, got)
			assert.Len(t, got.GetEntries(), 0)
		})
	}
}

func TestTransactionToProto_EntriesKeepOrder(t *testing.T) {
	tx := newTestTransaction()
	tx.Entries = []models.Entry{
		newTestEntry(TEST_ENTRY_ID_CREDIT, TEST_TO_ACCOUNT_ID, models.DIRECTION_CREDIT),
		newTestEntry(TEST_ENTRY_ID_DEBIT, TEST_FROM_ACCOUNT_ID, models.DIRECTION_DEBIT),
		newTestEntry(TEST_ENTRY_ID_THIRD, TEST_ACCOUNT_ID, models.DIRECTION_CREDIT),
	}
	want := []*ledgerv1.Entry{
		{Id: TEST_ENTRY_ID_CREDIT, AccountId: TEST_TO_ACCOUNT_ID, Direction: ledgerv1.EntryDirection_ENTRY_DIRECTION_CREDIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB},
		{Id: TEST_ENTRY_ID_DEBIT, AccountId: TEST_FROM_ACCOUNT_ID, Direction: ledgerv1.EntryDirection_ENTRY_DIRECTION_DEBIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB},
		{Id: TEST_ENTRY_ID_THIRD, AccountId: TEST_ACCOUNT_ID, Direction: ledgerv1.EntryDirection_ENTRY_DIRECTION_CREDIT, Amount: TEST_AMOUNT, Currency: TEST_CURRENCY_RUB},
	}

	got := transactionToProto(tx)

	require.NotNil(t, got)
	require.Len(t, got.GetEntries(), len(want))
	for i := range want {
		assert.Truef(t, proto.Equal(want[i], got.GetEntries()[i]), "entry %d: want %v, got %v", i, want[i], got.GetEntries()[i])
	}
}

func TestTransactionToProto_CreatedAtNanoseconds(t *testing.T) {
	createdAt := time.Date(2026, time.March, 4, 5, 6, 7, 123456789, time.UTC)
	tx := newTestTransaction()
	tx.CreatedAt = createdAt

	got := transactionToProto(tx)

	require.NotNil(t, got)
	require.NotNil(t, got.GetCreatedAt())
	assert.True(t, createdAt.Equal(got.GetCreatedAt().AsTime()))
}

func TestTransactionsToProto(t *testing.T) {
	first := newTestTransaction()
	second := newTestTransaction()
	second.ID = uuid.MustParse(TEST_TRANSACTION_ID_2)
	second.Type = models.TYPE_DEPOSIT
	third := newTestTransaction()
	third.ID = uuid.MustParse(TEST_TRANSACTION_ID_3)
	third.Status = models.STATUS_DECLINED

	tests := []struct {
		name         string
		transactions []*models.Transaction
	}{
		{name: "three transactions", transactions: []*models.Transaction{first, second, third}},
		{name: "nil slice", transactions: nil},
		{name: "empty slice", transactions: []*models.Transaction{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transactionsToProto(tt.transactions)

			require.Len(t, got, len(tt.transactions))
			for i, tx := range tt.transactions {
				want := transactionToProto(tx)
				assert.Truef(t, proto.Equal(want, got[i]), "transaction %d: want %v, got %v", i, want, got[i])
			}
		})
	}
}

// newTestEntry - проводка тестовой транзакции; TransactionID и CreatedAt заданы, чтобы проверить, что они не уходят наружу.
func newTestEntry(entryID, accountID string, direction models.Direction) models.Entry {
	return models.Entry{
		ID:            uuid.MustParse(entryID),
		TransactionID: uuid.MustParse(TEST_TRANSACTION_ID),
		AccountID:     uuid.MustParse(accountID),
		Direction:     direction,
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY_RUB,
		CreatedAt:     testCreatedAt(),
	}
}
