package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

// Неизменяемые sentinel-ошибки зависимостей
var (
	errDB        = errors.New("db failure")
	errTransport = errors.New("transport failure")
)

var (
	_ interfaces.AccountRepository = (*fakeAccountRepository)(nil)
	_ interfaces.Customers         = (*fakeCustomers)(nil)
	_ interfaces.Ledger            = (*fakeLedger)(nil)
)

type createCall struct {
	Account        models.Account
	MaxPerCurrency int
}

type getByIDCall struct {
	UserID    *uuid.UUID
	AccountID uuid.UUID
}

type listCall struct {
	UserID uuid.UUID
	Page   models.Page
}

type historyCall struct {
	AccountID uuid.UUID
	Page      models.Page
}

type transitionCall struct {
	AccountID   uuid.UUID
	Transition  models.Transition
	AllowedFrom []models.AccountStatus
}

type getBalanceCall struct {
	AccountID uuid.UUID
	Currency  models.Currency
}

// fakeAccountRepository отдаёт заданные значения и записывает вызовы
type fakeAccountRepository struct {
	mu sync.Mutex

	nextSerial       int64
	nextSerialErr    error
	createResult     *models.Account
	createErr        error
	getByIDResult    *models.Account
	getByIDErr       error
	listResult       []*models.Account
	listErr          error
	historyResult    []*models.StatusChange
	historyErr       error
	transitionResult *models.Account
	transitionErr    error

	nextSerialCalls int
	createCalls     []createCall
	getByIDCalls    []getByIDCall
	listCalls       []listCall
	historyCalls    []historyCall
	transitionCalls []transitionCall
}

func (f *fakeAccountRepository) NextSerial(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextSerialCalls++
	if f.nextSerialErr != nil {
		return 0, f.nextSerialErr
	}
	return f.nextSerial, nil
}

func (f *fakeAccountRepository) Create(_ context.Context, account models.Account, maxPerCurrency int) (*models.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls = append(f.createCalls, createCall{Account: account, MaxPerCurrency: maxPerCurrency})
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createResult, nil
}

func (f *fakeAccountRepository) GetByID(_ context.Context, userID *uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var userCopy *uuid.UUID
	if userID != nil {
		u := *userID
		userCopy = &u
	}
	f.getByIDCalls = append(f.getByIDCalls, getByIDCall{UserID: userCopy, AccountID: accountID})
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.getByIDResult, nil
}

func (f *fakeAccountRepository) ListByUser(_ context.Context, userID uuid.UUID, page models.Page) ([]*models.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls = append(f.listCalls, listCall{UserID: userID, Page: page})
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeAccountRepository) ListHistory(_ context.Context, accountID uuid.UUID, page models.Page) ([]*models.StatusChange, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.historyCalls = append(f.historyCalls, historyCall{AccountID: accountID, Page: page})
	if f.historyErr != nil {
		return nil, f.historyErr
	}
	return f.historyResult, nil
}

func (f *fakeAccountRepository) Transition(_ context.Context, accountID uuid.UUID, t models.Transition, allowedFrom []models.AccountStatus) (*models.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transitionCalls = append(f.transitionCalls, transitionCall{
		AccountID:   accountID,
		Transition:  t,
		AllowedFrom: append([]models.AccountStatus(nil), allowedFrom...),
	})
	if f.transitionErr != nil {
		return nil, f.transitionErr
	}
	return f.transitionResult, nil
}

// fakeCustomers отдаёт заданный признак активности и записывает userID
type fakeCustomers struct {
	mu     sync.Mutex
	active bool
	err    error
	calls  []uuid.UUID
}

func (f *fakeCustomers) IsActive(_ context.Context, userID uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, userID)
	if f.err != nil {
		return false, f.err
	}
	return f.active, nil
}

// fakeLedger отдаёт заданный остаток и записывает вызовы
type fakeLedger struct {
	mu     sync.Mutex
	result *models.Balance
	err    error
	calls  []getBalanceCall
}

func (f *fakeLedger) GetBalance(_ context.Context, accountID uuid.UUID, currency models.Currency) (*models.Balance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, getBalanceCall{AccountID: accountID, Currency: currency})
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// newTestService собирает сервис на настоящих CurrencySet и NumberGenerator
func newTestService(t *testing.T, repo *fakeAccountRepository, customers *fakeCustomers, ledger *fakeLedger, defaultCurrency models.Currency) *AccountService {
	t.Helper()
	currencies, err := models.NewCurrencySet([]string{TEST_CURRENCY_RUB, TEST_CURRENCY_USD, TEST_CURRENCY_EUR})
	require.NoError(t, err)
	numbers, err := models.NewNumberGenerator(TEST_BALANCE_PREFIX)
	require.NoError(t, err)
	repos := &repository.Repository{Account: repo}
	clients := &client.Clients{Customer: customers, Ledger: ledger}
	return NewAccountService(repos, clients, currencies, defaultCurrency, numbers)
}

// assertAllGetByIDCalls проверяет, что был хотя бы один вызов GetByID и все с ожидаемыми аргументами
func assertAllGetByIDCalls(t *testing.T, calls []getByIDCall, want getByIDCall) {
	t.Helper()
	require.NotEmpty(t, calls)
	for _, call := range calls {
		assert.Equal(t, want, call)
	}
}
