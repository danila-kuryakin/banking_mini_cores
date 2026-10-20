package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

var (
	TEST_USER1_ID       = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_USER2_ID       = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TEST_OFFICER_ID     = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	TEST_ACCOUNT_A_ID   = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	TEST_ACCOUNT_B_ID   = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	TEST_ACCOUNT_C_ID   = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	TEST_CASH_RUB_ID    = uuid.MustParse("00000000-0000-0000-0000-000000000643")
	TEST_CASH_USD_ID    = uuid.MustParse("00000000-0000-0000-0000-000000000840")
	TEST_TX_ID          = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	TEST_DECLINED_TX_ID = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

const (
	TEST_CHECK_ID     = "chk-1"
	TEST_TRANSFER_SUM = int64(1500)
)

// Номера счетов A, B, C и номер, которого нет в account-service.
const (
	TEST_NUMBER_A       = "40817810100000000001"
	TEST_NUMBER_B       = "40817810200000000002"
	TEST_NUMBER_C       = "40817840300000000003"
	TEST_NUMBER_MISSING = "40817810900000000099"
)

// declineCall - записанный вызов Decline
type declineCall struct {
	Posting models.Posting
	Reason  string
}

// listCall - записанный вызов ListTransactions
type listCall struct {
	AccountID uuid.UUID
	Page      models.Page
}

// fakeLedgerRepository - in-memory репозиторий леджера с настраиваемыми ответами
type fakeLedgerRepository struct {
	postTx     *models.Transaction
	postErr    error
	declineTx  *models.Transaction
	declineErr error

	transactions map[uuid.UUID]*models.Transaction
	getTxErr     error
	listResult   []*models.Transaction
	listErr      error
	balances     map[uuid.UUID]*models.Balance
	balanceErr   error

	ensureCalls  int
	postCalls    []models.Posting
	declineCalls []declineCall
	getTxCalls   []uuid.UUID
	listCalls    []listCall
	balanceCalls []uuid.UUID
}

var _ interfaces.LedgerRepository = (*fakeLedgerRepository)(nil)

func (f *fakeLedgerRepository) EnsureCashAccounts(_ context.Context, _ []models.Currency) (map[models.Currency]uuid.UUID, error) {
	f.ensureCalls++
	return nil, errors.New("unexpected call")
}

func (f *fakeLedgerRepository) Post(_ context.Context, p models.Posting) (*models.Transaction, error) {
	f.postCalls = append(f.postCalls, p)
	return f.postTx, f.postErr
}

func (f *fakeLedgerRepository) Decline(_ context.Context, p models.Posting, reason string) (*models.Transaction, error) {
	f.declineCalls = append(f.declineCalls, declineCall{Posting: p, Reason: reason})
	if f.declineErr != nil {
		return nil, f.declineErr
	}
	return f.declineTx, nil
}

func (f *fakeLedgerRepository) GetTransaction(_ context.Context, transactionID uuid.UUID) (*models.Transaction, error) {
	f.getTxCalls = append(f.getTxCalls, transactionID)
	if f.getTxErr != nil {
		return nil, f.getTxErr
	}
	tx, ok := f.transactions[transactionID]
	if !ok {
		return nil, domain.ErrTransactionNotFound
	}
	return tx, nil
}

func (f *fakeLedgerRepository) ListTransactions(_ context.Context, accountID uuid.UUID, page models.Page) ([]*models.Transaction, error) {
	f.listCalls = append(f.listCalls, listCall{AccountID: accountID, Page: page})
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeLedgerRepository) GetBalance(_ context.Context, accountID uuid.UUID) (*models.Balance, error) {
	f.balanceCalls = append(f.balanceCalls, accountID)
	if f.balanceErr != nil {
		return nil, f.balanceErr
	}
	balance, ok := f.balances[accountID]
	if !ok {
		return nil, domain.ErrBalanceNotFound
	}
	return balance, nil
}

// fakeAccounts - справочник счетов account-service: счёт по id и ошибка на конкретный id
type fakeAccounts struct {
	accounts map[uuid.UUID]*models.AccountInfo
	errs     map[uuid.UUID]error
	calls    []uuid.UUID

	numbers     map[string]uuid.UUID
	numberErr   error
	numberCalls []string
}

var _ interfaces.Accounts = (*fakeAccounts)(nil)

func (f *fakeAccounts) GetAccount(_ context.Context, accountID uuid.UUID) (*models.AccountInfo, error) {
	f.calls = append(f.calls, accountID)
	if err, ok := f.errs[accountID]; ok {
		return nil, err
	}
	account, ok := f.accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("account %s: %w", accountID, domain.ErrAccountNotFound)
	}
	return account, nil
}

func (f *fakeAccounts) GetAccountByNumber(_ context.Context, number string) (*models.AccountInfo, error) {
	f.numberCalls = append(f.numberCalls, number)
	if f.numberErr != nil {
		return nil, f.numberErr
	}
	account, ok := f.accounts[f.numbers[number]]
	if !ok {
		return nil, fmt.Errorf("number %s: %w", number, domain.ErrAccountNotFound)
	}
	return account, nil
}

// fakeFraudChecker - антифрод с заданным вердиктом или ошибкой
type fakeFraudChecker struct {
	verdict models.Verdict
	err     error
	calls   []models.TransferCommand
}

var _ interfaces.FraudChecker = (*fakeFraudChecker)(nil)

func (f *fakeFraudChecker) CheckTransfer(_ context.Context, cmd models.TransferCommand) (models.Verdict, error) {
	f.calls = append(f.calls, cmd)
	if f.err != nil {
		return models.Verdict{}, f.err
	}
	return f.verdict, nil
}

// testCash - кассовые счета для набора RUB/USD
func testCash() map[models.Currency]uuid.UUID {
	return map[models.Currency]uuid.UUID{"RUB": TEST_CASH_RUB_ID, "USD": TEST_CASH_USD_ID}
}

// newTestService собирает сервис с настоящим набором валют RUB/USD
func newTestService(t *testing.T, repo *fakeLedgerRepository, accounts *fakeAccounts, fraud *fakeFraudChecker, cash map[models.Currency]uuid.UUID) *LedgerService {
	t.Helper()
	currencies, err := models.NewCurrencySet([]string{"RUB", "USD"})
	require.NoError(t, err)
	return NewLedgerService(repo, accounts, fraud, currencies, cash)
}

// account строит информацию о счёте
func account(accountID, userID uuid.UUID, currency models.Currency, status models.AccountStatus) *models.AccountInfo {
	return &models.AccountInfo{ID: accountID, UserID: userID, Currency: currency, Status: status}
}

// accountsOf строит фейк accounts из списка счетов
func accountsOf(list ...*models.AccountInfo) *fakeAccounts {
	accounts := &fakeAccounts{accounts: map[uuid.UUID]*models.AccountInfo{}, errs: map[uuid.UUID]error{}, numbers: map[string]uuid.UUID{}}
	for _, info := range list {
		accounts.accounts[info.ID] = info
	}
	return accounts
}

// ptr возвращает указатель на копию значения
func ptr[T any](v T) *T {
	return &v
}
