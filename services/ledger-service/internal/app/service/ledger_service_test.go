package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

// completedTx - проведённый перевод A -> B, который отдаёт Post
func completedTx() *models.Transaction {
	return &models.Transaction{
		ID:            TEST_TX_ID,
		Type:          models.TYPE_TRANSFER,
		Status:        models.STATUS_COMPLETED,
		FromAccountID: TEST_ACCOUNT_A_ID,
		ToAccountID:   TEST_ACCOUNT_B_ID,
		Amount:        TEST_TRANSFER_SUM,
		Currency:      "RUB",
		InitiatorID:   ptr(TEST_USER1_ID),
	}
}

// declinedTx - отклонённый перевод A -> B, который отдают Decline и Post при нехватке средств
func declinedTx() *models.Transaction {
	return &models.Transaction{
		ID:            TEST_DECLINED_TX_ID,
		Type:          models.TYPE_TRANSFER,
		Status:        models.STATUS_DECLINED,
		FromAccountID: TEST_ACCOUNT_A_ID,
		ToAccountID:   TEST_ACCOUNT_B_ID,
		Amount:        TEST_TRANSFER_SUM,
		Currency:      "RUB",
		InitiatorID:   ptr(TEST_USER1_ID),
		DeclineReason: domain.DECLINE_ANTIFRAUD,
	}
}

// transferPosting - ожидаемая проводка перевода A(U1) -> B(U2) в RUB
func transferPosting(amount int64) models.Posting {
	return models.Posting{
		Type:        models.TYPE_TRANSFER,
		From:        TEST_ACCOUNT_A_ID,
		To:          TEST_ACCOUNT_B_ID,
		Amount:      amount,
		Currency:    "RUB",
		InitiatorID: ptr(TEST_USER1_ID),
		FromUserID:  ptr(TEST_USER1_ID),
		ToUserID:    ptr(TEST_USER2_ID),
	}
}

// transferCmd - перевод A -> B от U1 в указанной валюте
func transferCmd(currency string) models.TransferCommand {
	return models.TransferCommand{
		InitiatorID: TEST_USER1_ID,
		From:        TEST_ACCOUNT_A_ID,
		To:          TEST_ACCOUNT_B_ID,
		Amount:      TEST_TRANSFER_SUM,
		Currency:    currency,
	}
}

// defaultAccounts - A(U1, RUB), B(U2, RUB), C(U2, USD), все active
func defaultAccounts() *fakeAccounts {
	return accountsOf(
		account(TEST_ACCOUNT_A_ID, TEST_USER1_ID, "RUB", models.ACCOUNT_STATUS_ACTIVE),
		account(TEST_ACCOUNT_B_ID, TEST_USER2_ID, "RUB", models.ACCOUNT_STATUS_ACTIVE),
		account(TEST_ACCOUNT_C_ID, TEST_USER2_ID, "USD", models.ACCOUNT_STATUS_ACTIVE),
	)
}

// defaultRepo - Post отдаёт проведённую транзакцию, Decline - отклонённую
func defaultRepo() *fakeLedgerRepository {
	return &fakeLedgerRepository{postTx: completedTx(), declineTx: declinedTx()}
}

// errorIs строит проверку ошибки через ErrorIs
func errorIs(target error) require.ErrorAssertionFunc {
	return func(t require.TestingT, err error, _ ...interface{}) {
		require.ErrorIs(t, err, target)
	}
}

func TestLedgerService_Transfer_InvalidRequest(t *testing.T) {
	tests := []struct {
		name    string
		cmd     models.TransferCommand
		wantErr error
	}{
		{name: "currency_not_in_set", cmd: transferCmd("EUR"), wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency_two_letters", cmd: transferCmd("RU"), wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency_empty", cmd: transferCmd(""), wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency_spaces_only", cmd: transferCmd("   "), wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency_four_letters", cmd: transferCmd("RUBL"), wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency_with_digit", cmd: transferCmd("R1B"), wantErr: domain.ErrCurrencyNotAllowed},
		{
			name: "same_account",
			cmd: models.TransferCommand{
				InitiatorID: TEST_USER1_ID, From: TEST_ACCOUNT_A_ID, To: TEST_ACCOUNT_A_ID,
				Amount: TEST_TRANSFER_SUM, Currency: "RUB",
			},
			wantErr: domain.ErrSameAccount,
		},
		{
			// валюта проверяется раньше совпадения счетов
			name: "same_account_and_bad_currency",
			cmd: models.TransferCommand{
				InitiatorID: TEST_USER1_ID, From: TEST_ACCOUNT_A_ID, To: TEST_ACCOUNT_A_ID,
				Amount: TEST_TRANSFER_SUM, Currency: "EUR",
			},
			wantErr: domain.ErrCurrencyNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			accounts := defaultAccounts()
			fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: true}}
			svc := newTestService(t, repo, accounts, fraud, testCash())

			got, err := svc.Transfer(t.Context(), tt.cmd)

			assert.Nil(t, got)
			assert.Empty(t, accounts.calls)
			assert.Empty(t, fraud.calls)
			assert.Empty(t, repo.postCalls)
			assert.Empty(t, repo.declineCalls)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLedgerService_Transfer(t *testing.T) {
	errAccounts := errors.New("accounts unavailable")
	errDB := errors.New("db failure")

	tests := []struct {
		name            string
		cmd             models.TransferCommand
		setup           func(repo *fakeLedgerRepository, accounts *fakeAccounts, fraud *fakeFraudChecker)
		want            *models.Transaction
		wantErr         error
		wantFraudCalls  int
		wantPostCalls   []models.Posting
		wantDeclineCall []declineCall
	}{
		{
			// получатель B принадлежит другому клиенту - это норма
			name:           "success_lowercase_currency_other_owner",
			cmd:            transferCmd("rub"),
			setup:          func(*fakeLedgerRepository, *fakeAccounts, *fakeFraudChecker) {},
			want:           completedTx(),
			wantFraudCalls: 1,
			wantPostCalls:  []models.Posting{transferPosting(TEST_TRANSFER_SUM)},
		},
		{
			name: "success_amount_one",
			cmd: models.TransferCommand{
				InitiatorID: TEST_USER1_ID, From: TEST_ACCOUNT_A_ID, To: TEST_ACCOUNT_B_ID, Amount: 1, Currency: "RUB",
			},
			setup:          func(*fakeLedgerRepository, *fakeAccounts, *fakeFraudChecker) {},
			want:           completedTx(),
			wantFraudCalls: 1,
			wantPostCalls:  []models.Posting{transferPosting(1)},
		},
		{
			name: "source_not_found",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				delete(accounts.accounts, TEST_ACCOUNT_A_ID)
			},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name: "destination_not_found",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				delete(accounts.accounts, TEST_ACCOUNT_B_ID)
			},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name: "source_accounts_error",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.errs[TEST_ACCOUNT_A_ID] = errAccounts
			},
			wantErr: errAccounts,
		},
		{
			name: "destination_accounts_error",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.errs[TEST_ACCOUNT_B_ID] = errAccounts
			},
			wantErr: errAccounts,
		},
		{
			name: "source_blocked",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "source_closed",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = models.ACCOUNT_STATUS_CLOSED
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "source_empty_status",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = ""
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "destination_blocked",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "destination_closed",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_CLOSED
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "destination_empty_status",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = ""
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "destination_currency_mismatch",
			cmd: models.TransferCommand{
				InitiatorID: TEST_USER1_ID, From: TEST_ACCOUNT_A_ID, To: TEST_ACCOUNT_C_ID,
				Amount: TEST_TRANSFER_SUM, Currency: "RUB",
			},
			setup:   func(*fakeLedgerRepository, *fakeAccounts, *fakeFraudChecker) {},
			wantErr: domain.ErrCurrencyMismatch,
		},
		{
			name: "source_currency_mismatch",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Currency = "USD"
			},
			wantErr: domain.ErrCurrencyMismatch,
		},
		{
			name: "source_not_owned",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].UserID = TEST_USER2_ID
			},
			wantErr: domain.ErrAccountNotOwned,
		},
		{
			// источник проверяется полностью раньше получателя
			name: "source_not_owned_and_destination_blocked",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].UserID = TEST_USER2_ID
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			wantErr: domain.ErrAccountNotOwned,
		},
		{
			// источник проверяется полностью раньше получателя
			name: "source_blocked_and_destination_not_found",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts, _ *fakeFraudChecker) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = models.ACCOUNT_STATUS_BLOCKED
				delete(accounts.accounts, TEST_ACCOUNT_B_ID)
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "fraud_check_error",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, _ *fakeAccounts, fraud *fakeFraudChecker) {
				fraud.err = errors.New("antifraud unavailable")
			},
			wantErr:        domain.ErrFraudCheckFailed,
			wantFraudCalls: 1,
		},
		{
			name: "deny_with_check_id",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, _ *fakeAccounts, fraud *fakeFraudChecker) {
				fraud.verdict = models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID}
			},
			want:           declinedTx(),
			wantErr:        domain.ErrTransferDenied,
			wantFraudCalls: 1,
			wantDeclineCall: []declineCall{
				{Posting: transferPosting(TEST_TRANSFER_SUM), Reason: "denied by antifraud (check chk-1)"},
			},
		},
		{
			name: "deny_without_check_id",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, _ *fakeAccounts, fraud *fakeFraudChecker) {
				fraud.verdict = models.Verdict{Allowed: false}
			},
			want:           declinedTx(),
			wantErr:        domain.ErrTransferDenied,
			wantFraudCalls: 1,
			wantDeclineCall: []declineCall{
				{Posting: transferPosting(TEST_TRANSFER_SUM), Reason: "denied by antifraud"},
			},
		},
		{
			name: "deny_with_empty_reasons",
			cmd:  transferCmd("RUB"),
			setup: func(_ *fakeLedgerRepository, _ *fakeAccounts, fraud *fakeFraudChecker) {
				fraud.verdict = models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID, Reasons: []string{}}
			},
			want:           declinedTx(),
			wantErr:        domain.ErrTransferDenied,
			wantFraudCalls: 1,
			wantDeclineCall: []declineCall{
				{Posting: transferPosting(TEST_TRANSFER_SUM), Reason: "denied by antifraud (check chk-1)"},
			},
		},
		{
			name: "post_insufficient_funds",
			cmd:  transferCmd("RUB"),
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts, _ *fakeFraudChecker) {
				repo.postTx = declinedTx()
				repo.postErr = domain.ErrInsufficientFunds
			},
			want:           declinedTx(),
			wantErr:        domain.ErrInsufficientFunds,
			wantFraudCalls: 1,
			wantPostCalls:  []models.Posting{transferPosting(TEST_TRANSFER_SUM)},
		},
		{
			name: "post_db_error",
			cmd:  transferCmd("RUB"),
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts, _ *fakeFraudChecker) {
				repo.postTx = nil
				repo.postErr = errDB
			},
			wantErr:        errDB,
			wantFraudCalls: 1,
			wantPostCalls:  []models.Posting{transferPosting(TEST_TRANSFER_SUM)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			accounts := defaultAccounts()
			fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: true}}
			tt.setup(repo, accounts, fraud)
			svc := newTestService(t, repo, accounts, fraud, testCash())

			got, err := svc.Transfer(t.Context(), tt.cmd)

			assert.Len(t, fraud.calls, tt.wantFraudCalls)
			assert.Equal(t, tt.wantPostCalls, repo.postCalls)
			assert.Equal(t, tt.wantDeclineCall, repo.declineCalls)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLedgerService_Transfer_FraudCalledWithCommand(t *testing.T) {
	repo := defaultRepo()
	accounts := defaultAccounts()
	fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: true}}
	svc := newTestService(t, repo, accounts, fraud, testCash())

	_, err := svc.Transfer(t.Context(), transferCmd("rub"))
	require.NoError(t, err)

	require.Len(t, fraud.calls, 1)
	assert.Equal(t, TEST_USER1_ID, fraud.calls[0].InitiatorID)
	assert.Equal(t, TEST_ACCOUNT_A_ID, fraud.calls[0].From)
	assert.Equal(t, TEST_ACCOUNT_B_ID, fraud.calls[0].To)
	assert.Equal(t, TEST_TRANSFER_SUM, fraud.calls[0].Amount)
}

func TestLedgerService_Transfer_DeclineError(t *testing.T) {
	errDB := errors.New("db failure")
	repo := defaultRepo()
	repo.declineErr = errDB
	accounts := defaultAccounts()
	fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID}}
	svc := newTestService(t, repo, accounts, fraud, testCash())

	got, err := svc.Transfer(t.Context(), transferCmd("RUB"))

	assert.Nil(t, got)
	assert.Empty(t, repo.postCalls)
	require.ErrorIs(t, err, errDB)
	assert.NotErrorIs(t, err, domain.ErrTransferDenied)
}

func TestLedgerService_Deposit_InvalidCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
	}{
		{name: "not_in_set", currency: "EUR"},
		{name: "empty", currency: ""},
		{name: "spaces_only", currency: "  "},
		{name: "two_letters", currency: "RU"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			accounts := defaultAccounts()
			svc := newTestService(t, repo, accounts, &fakeFraudChecker{}, testCash())

			got, err := svc.Deposit(t.Context(), models.DepositCommand{
				AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: tt.currency, OfficerID: TEST_OFFICER_ID,
			})

			assert.Nil(t, got)
			assert.Empty(t, accounts.calls)
			assert.Empty(t, repo.postCalls)
			assert.Empty(t, repo.declineCalls)
			require.ErrorIs(t, err, domain.ErrCurrencyNotAllowed)
		})
	}
}

func TestLedgerService_Deposit(t *testing.T) {
	errAccounts := errors.New("accounts unavailable")
	errDB := errors.New("db failure")
	depositTx := &models.Transaction{
		ID:            TEST_TX_ID,
		Type:          models.TYPE_DEPOSIT,
		Status:        models.STATUS_COMPLETED,
		FromAccountID: TEST_CASH_USD_ID,
		ToAccountID:   TEST_ACCOUNT_C_ID,
		Amount:        500,
		Currency:      "USD",
		InitiatorID:   ptr(TEST_OFFICER_ID),
	}

	tests := []struct {
		name          string
		cmd           models.DepositCommand
		setup         func(repo *fakeLedgerRepository, accounts *fakeAccounts)
		want          *models.Transaction
		wantErr       error
		wantPostCalls []models.Posting
	}{
		{
			// счёт C принадлежит U2, вносит офицер - владелец не проверяется
			name: "success_usd_with_spaces",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_C_ID, Amount: 500, Currency: " usd ", OfficerID: TEST_OFFICER_ID},
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts) {
				repo.postTx = depositTx
			},
			want: depositTx,
			wantPostCalls: []models.Posting{{
				Type: models.TYPE_DEPOSIT, From: TEST_CASH_USD_ID, To: TEST_ACCOUNT_C_ID, Amount: 500, Currency: "USD",
				InitiatorID: ptr(TEST_OFFICER_ID), FromUserID: nil, ToUserID: ptr(TEST_USER2_ID),
			}},
		},
		{
			name: "success_rub_cash",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 700, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts) {
				repo.postTx = depositTx
			},
			want: depositTx,
			wantPostCalls: []models.Posting{{
				Type: models.TYPE_DEPOSIT, From: TEST_CASH_RUB_ID, To: TEST_ACCOUNT_B_ID, Amount: 700, Currency: "RUB",
				InitiatorID: ptr(TEST_OFFICER_ID), FromUserID: nil, ToUserID: ptr(TEST_USER2_ID),
			}},
		},
		{
			name: "account_not_found",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				delete(accounts.accounts, TEST_ACCOUNT_B_ID)
			},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name: "accounts_error",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.errs[TEST_ACCOUNT_B_ID] = errAccounts
			},
			wantErr: errAccounts,
		},
		{
			name: "account_blocked",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "account_closed",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_CLOSED
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name: "account_empty_status",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = ""
			},
			wantErr: domain.ErrAccountNotActive,
		},
		{
			name:    "account_currency_mismatch",
			cmd:     models.DepositCommand{AccountID: TEST_ACCOUNT_C_ID, Amount: 500, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup:   func(*fakeLedgerRepository, *fakeAccounts) {},
			wantErr: domain.ErrCurrencyMismatch,
		},
		{
			name: "post_db_error",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 700, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts) {
				repo.postTx = nil
				repo.postErr = errDB
			},
			wantErr: errDB,
			wantPostCalls: []models.Posting{{
				Type: models.TYPE_DEPOSIT, From: TEST_CASH_RUB_ID, To: TEST_ACCOUNT_B_ID, Amount: 700, Currency: "RUB",
				InitiatorID: ptr(TEST_OFFICER_ID), FromUserID: nil, ToUserID: ptr(TEST_USER2_ID),
			}},
		},
		{
			// результат Post пробрасывается как есть, даже вместе с ошибкой
			name: "post_returns_tx_with_error",
			cmd:  models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 700, Currency: "RUB", OfficerID: TEST_OFFICER_ID},
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts) {
				repo.postTx = declinedTx()
				repo.postErr = domain.ErrInsufficientFunds
			},
			want:    declinedTx(),
			wantErr: domain.ErrInsufficientFunds,
			wantPostCalls: []models.Posting{{
				Type: models.TYPE_DEPOSIT, From: TEST_CASH_RUB_ID, To: TEST_ACCOUNT_B_ID, Amount: 700, Currency: "RUB",
				InitiatorID: ptr(TEST_OFFICER_ID), FromUserID: nil, ToUserID: ptr(TEST_USER2_ID),
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			accounts := defaultAccounts()
			tt.setup(repo, accounts)
			svc := newTestService(t, repo, accounts, &fakeFraudChecker{}, testCash())

			got, err := svc.Deposit(t.Context(), tt.cmd)

			assert.Equal(t, tt.wantPostCalls, repo.postCalls)
			assert.Empty(t, repo.declineCalls)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLedgerService_Deposit_NoCashAccount(t *testing.T) {
	tests := []struct {
		name     string
		cash     map[models.Currency]uuid.UUID
		currency string
		account  uuid.UUID
	}{
		{name: "no_cash_for_usd", cash: map[models.Currency]uuid.UUID{"RUB": TEST_CASH_RUB_ID}, currency: "USD", account: TEST_ACCOUNT_C_ID},
		{name: "empty_cash_map", cash: map[models.Currency]uuid.UUID{}, currency: "RUB", account: TEST_ACCOUNT_B_ID},
		{name: "nil_cash_map", cash: nil, currency: "RUB", account: TEST_ACCOUNT_B_ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			svc := newTestService(t, repo, defaultAccounts(), &fakeFraudChecker{}, tt.cash)

			got, err := svc.Deposit(t.Context(), models.DepositCommand{
				AccountID: tt.account, Amount: 500, Currency: tt.currency, OfficerID: TEST_OFFICER_ID,
			})

			assert.Nil(t, got)
			assert.Empty(t, repo.postCalls)
			assert.Empty(t, repo.declineCalls)
			// у этой ошибки нет sentinel (R13)
			require.Error(t, err)
		})
	}
}

func TestLedgerService_GetTransaction(t *testing.T) {
	errDB := errors.New("db failure")

	tests := []struct {
		name    string
		setup   func(repo *fakeLedgerRepository)
		want    *models.Transaction
		wantErr error
	}{
		{
			name: "found",
			setup: func(repo *fakeLedgerRepository) {
				repo.transactions = map[uuid.UUID]*models.Transaction{TEST_TX_ID: completedTx()}
			},
			want: completedTx(),
		},
		{
			name:    "not_found",
			setup:   func(*fakeLedgerRepository) {},
			wantErr: domain.ErrTransactionNotFound,
		},
		{
			name: "db_error",
			setup: func(repo *fakeLedgerRepository) {
				repo.getTxErr = errDB
			},
			wantErr: errDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			tt.setup(repo)
			svc := newTestService(t, repo, defaultAccounts(), &fakeFraudChecker{}, testCash())

			got, err := svc.GetTransaction(t.Context(), TEST_TX_ID)

			assert.Equal(t, []uuid.UUID{TEST_TX_ID}, repo.getTxCalls)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLedgerService_GetBalance(t *testing.T) {
	errDB := errors.New("db failure")

	tests := []struct {
		name    string
		setup   func(repo *fakeLedgerRepository)
		want    *models.Balance
		wantErr error
	}{
		{
			name: "found",
			setup: func(repo *fakeLedgerRepository) {
				repo.balances = map[uuid.UUID]*models.Balance{
					TEST_ACCOUNT_A_ID: {AccountID: TEST_ACCOUNT_A_ID, Amount: 1500, Currency: "RUB"},
				}
			},
			want: &models.Balance{AccountID: TEST_ACCOUNT_A_ID, Amount: 1500, Currency: "RUB"},
		},
		{
			name:    "no_movements",
			setup:   func(*fakeLedgerRepository) {},
			wantErr: domain.ErrBalanceNotFound,
		},
		{
			name: "db_error",
			setup: func(repo *fakeLedgerRepository) {
				repo.balanceErr = errDB
			},
			wantErr: errDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			tt.setup(repo)
			svc := newTestService(t, repo, defaultAccounts(), &fakeFraudChecker{}, testCash())

			got, err := svc.GetBalance(t.Context(), TEST_ACCOUNT_A_ID)

			assert.Equal(t, []uuid.UUID{TEST_ACCOUNT_A_ID}, repo.balanceCalls)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLedgerService_ListTransactions(t *testing.T) {
	errAccounts := errors.New("accounts unavailable")
	errDB := errors.New("db failure")
	statement := func() []*models.Transaction {
		return []*models.Transaction{completedTx(), declinedTx()}
	}

	tests := []struct {
		name             string
		userID           *uuid.UUID
		page             models.Page
		setup            func(repo *fakeLedgerRepository, accounts *fakeAccounts)
		want             []*models.Transaction
		wantErr          error
		wantAccountCalls []uuid.UUID
		wantRepoCalls    []listCall
	}{
		{
			name:          "no_owner_page_as_is",
			page:          models.Page{Limit: 10, Offset: 5},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 10, Offset: 5}}},
		},
		{
			name:          "zero_limit_negative_offset",
			page:          models.Page{Limit: 0, Offset: -3},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 20, Offset: 0}}},
		},
		{
			name:          "negative_limit",
			page:          models.Page{Limit: -1, Offset: 0},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 20, Offset: 0}}},
		},
		{
			name:          "limit_above_max",
			page:          models.Page{Limit: 101, Offset: 5},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 100, Offset: 5}}},
		},
		{
			name:          "limit_one",
			page:          models.Page{Limit: 1, Offset: 0},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 1, Offset: 0}}},
		},
		{
			name:          "limit_max",
			page:          models.Page{Limit: 100, Offset: 7},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 100, Offset: 7}}},
		},
		{
			name:          "offset_minus_one",
			page:          models.Page{Limit: 20, Offset: -1},
			setup:         func(*fakeLedgerRepository, *fakeAccounts) {},
			want:          statement(),
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 20, Offset: 0}}},
		},
		{
			name:             "own_account",
			userID:           ptr(TEST_USER1_ID),
			page:             models.Page{Limit: 0, Offset: 2},
			setup:            func(*fakeLedgerRepository, *fakeAccounts) {},
			want:             statement(),
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
			wantRepoCalls:    []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 20, Offset: 2}}},
		},
		{
			name:   "own_blocked_account",
			userID: ptr(TEST_USER1_ID),
			page:   models.Page{Limit: 10, Offset: 0},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			want:             statement(),
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
			wantRepoCalls:    []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 10, Offset: 0}}},
		},
		{
			name:   "own_closed_account",
			userID: ptr(TEST_USER1_ID),
			page:   models.Page{Limit: 10, Offset: 0},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = models.ACCOUNT_STATUS_CLOSED
			},
			want:             statement(),
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
			wantRepoCalls:    []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 10, Offset: 0}}},
		},
		{
			name:   "own_account_other_currency",
			userID: ptr(TEST_USER1_ID),
			page:   models.Page{Limit: 10, Offset: 0},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Currency = "EUR"
			},
			want:             statement(),
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
			wantRepoCalls:    []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 10, Offset: 0}}},
		},
		{
			// чужой счёт неотличим от несуществующего
			name:             "foreign_account",
			userID:           ptr(TEST_USER2_ID),
			page:             models.Page{Limit: 10, Offset: 0},
			setup:            func(*fakeLedgerRepository, *fakeAccounts) {},
			wantErr:          domain.ErrAccountNotFound,
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			name:   "account_not_found",
			userID: ptr(TEST_USER1_ID),
			page:   models.Page{Limit: 10, Offset: 0},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				delete(accounts.accounts, TEST_ACCOUNT_A_ID)
			},
			wantErr:          domain.ErrAccountNotFound,
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			name:   "accounts_error",
			userID: ptr(TEST_USER1_ID),
			page:   models.Page{Limit: 10, Offset: 0},
			setup: func(_ *fakeLedgerRepository, accounts *fakeAccounts) {
				accounts.errs[TEST_ACCOUNT_A_ID] = errAccounts
			},
			wantErr:          errAccounts,
			wantAccountCalls: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			name: "empty_statement",
			page: models.Page{Limit: 10, Offset: 0},
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts) {
				repo.listResult = []*models.Transaction{}
			},
			want:          []*models.Transaction{},
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 10, Offset: 0}}},
		},
		{
			name: "repo_error",
			page: models.Page{Limit: 10, Offset: 0},
			setup: func(repo *fakeLedgerRepository, _ *fakeAccounts) {
				repo.listErr = errDB
			},
			wantErr:       errDB,
			wantRepoCalls: []listCall{{AccountID: TEST_ACCOUNT_A_ID, Page: models.Page{Limit: 10, Offset: 0}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			repo.listResult = statement()
			accounts := defaultAccounts()
			tt.setup(repo, accounts)
			svc := newTestService(t, repo, accounts, &fakeFraudChecker{}, testCash())

			got, err := svc.ListTransactions(t.Context(), tt.userID, TEST_ACCOUNT_A_ID, tt.page)

			assert.Equal(t, tt.wantAccountCalls, accounts.calls)
			assert.Equal(t, tt.wantRepoCalls, repo.listCalls)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLedgerService_NoPostingOnPreCheckFailure(t *testing.T) {
	tests := []struct {
		name      string
		cash      map[models.Currency]uuid.UUID
		setup     func(accounts *fakeAccounts)
		call      func(ctx context.Context, svc *LedgerService) error
		assertErr require.ErrorAssertionFunc
	}{
		{
			name:  "transfer_currency_not_allowed",
			cash:  testCash(),
			setup: func(*fakeAccounts) {},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Transfer(ctx, transferCmd("EUR"))
				return err
			},
			assertErr: errorIs(domain.ErrCurrencyNotAllowed),
		},
		{
			name:  "transfer_same_account",
			cash:  testCash(),
			setup: func(*fakeAccounts) {},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Transfer(ctx, models.TransferCommand{
					InitiatorID: TEST_USER1_ID, From: TEST_ACCOUNT_A_ID, To: TEST_ACCOUNT_A_ID, Amount: 1, Currency: "RUB",
				})
				return err
			},
			assertErr: errorIs(domain.ErrSameAccount),
		},
		{
			name: "transfer_source_not_found",
			cash: testCash(),
			setup: func(accounts *fakeAccounts) {
				delete(accounts.accounts, TEST_ACCOUNT_A_ID)
			},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Transfer(ctx, transferCmd("RUB"))
				return err
			},
			assertErr: errorIs(domain.ErrAccountNotFound),
		},
		{
			name: "transfer_source_blocked",
			cash: testCash(),
			setup: func(accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_A_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Transfer(ctx, transferCmd("RUB"))
				return err
			},
			assertErr: errorIs(domain.ErrAccountNotActive),
		},
		{
			name:  "transfer_currency_mismatch",
			cash:  testCash(),
			setup: func(*fakeAccounts) {},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Transfer(ctx, models.TransferCommand{
					InitiatorID: TEST_USER1_ID, From: TEST_ACCOUNT_A_ID, To: TEST_ACCOUNT_C_ID, Amount: 1, Currency: "RUB",
				})
				return err
			},
			assertErr: errorIs(domain.ErrCurrencyMismatch),
		},
		{
			name: "transfer_not_owned",
			cash: testCash(),
			setup: func(accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_A_ID].UserID = TEST_USER2_ID
			},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Transfer(ctx, transferCmd("RUB"))
				return err
			},
			assertErr: errorIs(domain.ErrAccountNotOwned),
		},
		{
			name:  "deposit_currency_not_allowed",
			cash:  testCash(),
			setup: func(*fakeAccounts) {},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Deposit(ctx, models.DepositCommand{AccountID: TEST_ACCOUNT_B_ID, Amount: 1, Currency: "EUR", OfficerID: TEST_OFFICER_ID})
				return err
			},
			assertErr: errorIs(domain.ErrCurrencyNotAllowed),
		},
		{
			name:  "deposit_currency_mismatch",
			cash:  testCash(),
			setup: func(*fakeAccounts) {},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Deposit(ctx, models.DepositCommand{AccountID: TEST_ACCOUNT_C_ID, Amount: 1, Currency: "RUB", OfficerID: TEST_OFFICER_ID})
				return err
			},
			assertErr: errorIs(domain.ErrCurrencyMismatch),
		},
		{
			name:  "deposit_no_cash_account",
			cash:  map[models.Currency]uuid.UUID{"RUB": TEST_CASH_RUB_ID},
			setup: func(*fakeAccounts) {},
			call: func(ctx context.Context, svc *LedgerService) error {
				_, err := svc.Deposit(ctx, models.DepositCommand{AccountID: TEST_ACCOUNT_C_ID, Amount: 1, Currency: "USD", OfficerID: TEST_OFFICER_ID})
				return err
			},
			// у отсутствия кассы нет sentinel (R13)
			assertErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			accounts := defaultAccounts()
			fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: true}}
			tt.setup(accounts)
			svc := newTestService(t, repo, accounts, fraud, tt.cash)

			err := tt.call(t.Context(), svc)

			tt.assertErr(t, err)
			assert.Empty(t, fraud.calls)
			assert.Empty(t, repo.postCalls)
			assert.Empty(t, repo.declineCalls)
		})
	}
}

func TestLedgerService_Transfer_CurrencyNormalized(t *testing.T) {
	tests := []struct {
		name     string
		currency string
	}{
		{name: "lowercase", currency: "rub"},
		{name: "leading_space", currency: " RUB"},
		{name: "mixed_case", currency: "Rub"},
		{name: "trailing_tab", currency: "RUB\t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: true}}
			svc := newTestService(t, repo, defaultAccounts(), fraud, testCash())

			_, err := svc.Transfer(t.Context(), transferCmd(tt.currency))
			require.NoError(t, err)

			require.Len(t, repo.postCalls, 1)
			assert.Equal(t, models.Currency("RUB"), repo.postCalls[0].Currency)
		})
	}
}

func TestLedgerService_Deposit_CurrencyNormalized(t *testing.T) {
	tests := []struct {
		name     string
		currency string
	}{
		{name: "lowercase", currency: "rub"},
		{name: "leading_space", currency: " RUB"},
		{name: "mixed_case", currency: "Rub"},
		{name: "trailing_tab", currency: "RUB\t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			svc := newTestService(t, repo, defaultAccounts(), &fakeFraudChecker{}, testCash())

			_, err := svc.Deposit(t.Context(), models.DepositCommand{
				AccountID: TEST_ACCOUNT_B_ID, Amount: 500, Currency: tt.currency, OfficerID: TEST_OFFICER_ID,
			})
			require.NoError(t, err)

			require.Len(t, repo.postCalls, 1)
			assert.Equal(t, models.Currency("RUB"), repo.postCalls[0].Currency)
		})
	}
}

// numberAccounts - defaultAccounts с номерами счетов A, B и C
func numberAccounts() *fakeAccounts {
	accounts := defaultAccounts()
	accounts.numbers[TEST_NUMBER_A] = TEST_ACCOUNT_A_ID
	accounts.numbers[TEST_NUMBER_B] = TEST_ACCOUNT_B_ID
	accounts.numbers[TEST_NUMBER_C] = TEST_ACCOUNT_C_ID
	return accounts
}

// transferByNumberCmd - перевод со счёта A от U1 на счёт с указанным номером
func transferByNumberCmd(number string) models.TransferCommand {
	return models.TransferCommand{
		InitiatorID: TEST_USER1_ID,
		From:        TEST_ACCOUNT_A_ID,
		Amount:      TEST_TRANSFER_SUM,
		Currency:    "RUB",
		ToNumber:    number,
	}
}

func TestLedgerService_TransferByNumber(t *testing.T) {
	errAccounts := errors.New("accounts unavailable")

	tests := []struct {
		name            string
		cmd             models.TransferCommand
		setup           func(accounts *fakeAccounts)
		want            *models.Transaction
		wantErr         error
		wantNotErr      error
		wantAccountCall []uuid.UUID
		wantPostCalls   []models.Posting
	}{
		{
			// получатель найден по номеру, повторно по id его не ищем
			name:            "success",
			cmd:             transferByNumberCmd(TEST_NUMBER_B),
			setup:           func(*fakeAccounts) {},
			want:            completedTx(),
			wantAccountCall: []uuid.UUID{TEST_ACCOUNT_A_ID},
			wantPostCalls:   []models.Posting{transferPosting(TEST_TRANSFER_SUM)},
		},
		{
			name:    "own_source_number",
			cmd:     transferByNumberCmd(TEST_NUMBER_A),
			setup:   func(*fakeAccounts) {},
			wantErr: domain.ErrSameAccount,
		},
		{
			name:    "recipient_not_found",
			cmd:     transferByNumberCmd(TEST_NUMBER_MISSING),
			setup:   func(*fakeAccounts) {},
			wantErr: domain.ErrAccountNotFound,
		},
		{
			name: "recipient_blocked",
			cmd:  transferByNumberCmd(TEST_NUMBER_B),
			setup: func(accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			wantErr:         domain.ErrAccountNotActive,
			wantAccountCall: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			name: "recipient_closed",
			cmd:  transferByNumberCmd(TEST_NUMBER_B),
			setup: func(accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_CLOSED
			},
			wantErr:         domain.ErrAccountNotActive,
			wantAccountCall: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			name:            "recipient_other_currency",
			cmd:             transferByNumberCmd(TEST_NUMBER_C),
			setup:           func(*fakeAccounts) {},
			wantErr:         domain.ErrCurrencyMismatch,
			wantAccountCall: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			// источник проверяется раньше статуса получателя, как и при переводе по id
			name: "source_not_owned_and_recipient_blocked",
			cmd:  transferByNumberCmd(TEST_NUMBER_B),
			setup: func(accounts *fakeAccounts) {
				accounts.accounts[TEST_ACCOUNT_A_ID].UserID = TEST_USER2_ID
				accounts.accounts[TEST_ACCOUNT_B_ID].Status = models.ACCOUNT_STATUS_BLOCKED
			},
			wantErr:         domain.ErrAccountNotOwned,
			wantAccountCall: []uuid.UUID{TEST_ACCOUNT_A_ID},
		},
		{
			name: "accounts_error",
			cmd:  transferByNumberCmd(TEST_NUMBER_B),
			setup: func(accounts *fakeAccounts) {
				accounts.numberErr = errAccounts
			},
			wantErr:    errAccounts,
			wantNotErr: domain.ErrAccountNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			accounts := numberAccounts()
			tt.setup(accounts)
			fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: true}}
			svc := newTestService(t, repo, accounts, fraud, testCash())

			got, err := svc.Transfer(t.Context(), tt.cmd)

			assert.Equal(t, []string{tt.cmd.ToNumber}, accounts.numberCalls)
			assert.Equal(t, tt.wantAccountCall, accounts.calls)
			assert.Equal(t, tt.wantPostCalls, repo.postCalls)
			assert.Empty(t, repo.declineCalls)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.wantNotErr != nil {
					assert.NotErrorIs(t, err, tt.wantNotErr)
				}
				assert.Empty(t, fraud.calls)
				return
			}
			require.NoError(t, err)
			require.Len(t, fraud.calls, 1)
			assert.Equal(t, TEST_ACCOUNT_B_ID, fraud.calls[0].To)
		})
	}
}

func TestLedgerService_TransferByNumber_DenyDeclinesResolvedRecipient(t *testing.T) {
	repo := defaultRepo()
	accounts := numberAccounts()
	fraud := &fakeFraudChecker{verdict: models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID}}
	svc := newTestService(t, repo, accounts, fraud, testCash())

	got, err := svc.Transfer(t.Context(), transferByNumberCmd(TEST_NUMBER_B))

	require.ErrorIs(t, err, domain.ErrTransferDenied)
	assert.Equal(t, declinedTx(), got)
	assert.Equal(t, []declineCall{
		{Posting: transferPosting(TEST_TRANSFER_SUM), Reason: "denied by antifraud (check chk-1)"},
	}, repo.declineCalls)
}
