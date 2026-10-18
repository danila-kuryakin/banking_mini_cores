package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

const (
	TEST_BALANCE_PREFIX = "40817"
	TEST_CURRENCY_RUB   = "RUB"
	TEST_CURRENCY_USD   = "USD"
	TEST_CURRENCY_EUR   = "EUR"
	TEST_REASON         = "fraud suspicion"
	TEST_COMMENT        = "verified"
	TEST_NUMBER_READ    = "TEST_READ"
	TEST_NUMBER_UPDATED = "TEST_UPDATED"
	TEST_NUMBER_CREATED = "TEST_CREATED"
)

var (
	TEST_USER_ID    = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	TEST_ACCOUNT_ID = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	TEST_OFFICER_ID = uuid.MustParse("33333333-3333-4333-8333-333333333333")
)

// testAccount строит счёт клиента с заданными номером, валютой и статусом
func testAccount(number string, currency models.Currency, accountStatus models.AccountStatus) *models.Account {
	moment := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &models.Account{
		ID:              TEST_ACCOUNT_ID,
		UserID:          TEST_USER_ID,
		Number:          number,
		Currency:        currency,
		Status:          accountStatus,
		StatusChangedAt: moment,
		CreatedAt:       moment,
		UpdatedAt:       moment,
	}
}

func userPtr(userID uuid.UUID) *uuid.UUID {
	return &userID
}

// assertOpenCreateCalls проверяет аргументы каждого вызова Create (их 0 или 1)
func assertOpenCreateCalls(t *testing.T, calls []createCall, wantNumber string, wantCurrency models.Currency) {
	t.Helper()
	for _, call := range calls {
		assert.Equal(t, domain.MAX_ACCOUNTS_PER_CURRENCY, call.MaxPerCurrency)
		assert.Equal(t, TEST_USER_ID, call.Account.UserID)
		assert.Equal(t, wantCurrency, call.Account.Currency)
		assert.Equal(t, wantNumber, call.Account.Number)
		assert.NotEqual(t, uuid.Nil, call.Account.ID)
		assert.Equal(t, uuid.Version(7), call.Account.ID.Version())
	}
}

func TestAccountService_Open(t *testing.T) {
	userCalls := []uuid.UUID{TEST_USER_ID}
	tests := []struct {
		name              string
		defaultCurrency   models.Currency
		rawCurrency       string
		serial            int64
		serialErr         error
		active            bool
		activeErr         error
		createErr         error
		wantCustomerCalls []uuid.UUID
		wantSerialCalls   int
		wantCreateCalls   int
		wantCurrency      models.Currency
		wantCreateNumber  string
		wantErr           error
	}{
		{name: "empty currency uses default RUB", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_RUB, wantCreateNumber: "40817810400000000001"},
		{name: "spaces currency uses default RUB", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "   ", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_RUB, wantCreateNumber: "40817810400000000001"},
		{name: "padded lower usd normalized", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: " usd ", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_USD, wantCreateNumber: "40817840700000000001"},
		{name: "lower eur normalized", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "eur", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_EUR, wantCreateNumber: "40817978300000000001"},
		{name: "empty currency uses default USD", defaultCurrency: TEST_CURRENCY_USD, rawCurrency: "", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_USD, wantCreateNumber: "40817840700000000001"},
		{name: "upper USD as is", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "USD", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_USD, wantCreateNumber: "40817840700000000001"},
		{name: "mixed case Usd normalized", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "Usd", serial: 1, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_USD, wantCreateNumber: "40817840700000000001"},
		// S = 38 + 69 = 107; 7*3 = 21 -> ключ 1
		{name: "max serial", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "USD", serial: 99999999999, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_USD, wantCreateNumber: "40817840199999999999"},
		{name: "currency GEL not allowed", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "GEL", serial: 1, active: true, wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency RU bad format", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "RU", serial: 1, active: true, wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency R1B bad format", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "R1B", serial: 1, active: true, wantErr: domain.ErrCurrencyNotAllowed},
		{name: "currency RUBL bad format", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "RUBL", serial: 1, active: true, wantErr: domain.ErrCurrencyNotAllowed},
		{name: "customer not found", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serial: 1, activeErr: domain.ErrCustomerNotFound, wantCustomerCalls: userCalls, wantErr: domain.ErrCustomerNotFound},
		{name: "customer transport error", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serial: 1, activeErr: errTransport, wantCustomerCalls: userCalls, wantErr: errTransport},
		{name: "customer not active", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serial: 1, active: false, wantCustomerCalls: userCalls, wantErr: domain.ErrCustomerNotActive},
		{name: "next serial db error", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serialErr: errDB, active: true, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantErr: errDB},
		{name: "create limit reached", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serial: 1, active: true, createErr: domain.ErrAccountLimitReached, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_RUB, wantCreateNumber: "40817810400000000001", wantErr: domain.ErrAccountLimitReached},
		{name: "create db error", defaultCurrency: TEST_CURRENCY_RUB, rawCurrency: "", serial: 1, active: true, createErr: errDB, wantCustomerCalls: userCalls, wantSerialCalls: 1, wantCreateCalls: 1, wantCurrency: TEST_CURRENCY_RUB, wantCreateNumber: "40817810400000000001", wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := testAccount(TEST_NUMBER_CREATED, tt.wantCurrency, models.STATUS_ACTIVE)
			repo := &fakeAccountRepository{nextSerial: tt.serial, nextSerialErr: tt.serialErr, createResult: created, createErr: tt.createErr}
			customers := &fakeCustomers{active: tt.active, err: tt.activeErr}
			svc := newTestService(t, repo, customers, &fakeLedger{}, tt.defaultCurrency)

			got, err := svc.Open(t.Context(), TEST_USER_ID, tt.rawCurrency)

			assert.Equal(t, tt.wantCustomerCalls, customers.calls)
			assert.Equal(t, tt.wantSerialCalls, repo.nextSerialCalls)
			require.Len(t, repo.createCalls, tt.wantCreateCalls)
			assertOpenCreateCalls(t, repo.createCalls, tt.wantCreateNumber, tt.wantCurrency)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, TEST_NUMBER_CREATED, got.Number)
		})
	}
}

func TestAccountService_Open_SerialOutOfRange(t *testing.T) {
	tests := []struct {
		name   string
		serial int64
	}{
		{name: "serial zero", serial: 0},
		{name: "serial above 11 digits", serial: 100000000000},
		{name: "serial negative", serial: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{nextSerial: tt.serial, createResult: testAccount(TEST_NUMBER_CREATED, TEST_CURRENCY_RUB, models.STATUS_ACTIVE)}
			svc := newTestService(t, repo, &fakeCustomers{active: true}, &fakeLedger{}, TEST_CURRENCY_RUB)

			got, err := svc.Open(t.Context(), TEST_USER_ID, "")

			require.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, 1, repo.nextSerialCalls)
			assert.Empty(t, repo.createCalls)
		})
	}
}

func TestAccountService_Open_ReturnsCreateResult(t *testing.T) {
	created := testAccount(TEST_NUMBER_CREATED, TEST_CURRENCY_RUB, models.STATUS_ACTIVE)
	repo := &fakeAccountRepository{nextSerial: 1, createResult: created}
	svc := newTestService(t, repo, &fakeCustomers{active: true}, &fakeLedger{}, TEST_CURRENCY_RUB)

	got, err := svc.Open(t.Context(), TEST_USER_ID, "")
	require.NoError(t, err)

	assert.Same(t, created, got)
}

func TestAccountService_Open_TwoCallsDistinctIDs(t *testing.T) {
	repo := &fakeAccountRepository{nextSerial: 1, createResult: testAccount(TEST_NUMBER_CREATED, TEST_CURRENCY_RUB, models.STATUS_ACTIVE)}
	svc := newTestService(t, repo, &fakeCustomers{active: true}, &fakeLedger{}, TEST_CURRENCY_RUB)

	_, err := svc.Open(t.Context(), TEST_USER_ID, "")
	require.NoError(t, err)
	_, err = svc.Open(t.Context(), TEST_USER_ID, "")
	require.NoError(t, err)

	require.Len(t, repo.createCalls, 2)
	assert.NotEqual(t, repo.createCalls[0].Account.ID, repo.createCalls[1].Account.ID)
	assert.Equal(t, uuid.Version(7), repo.createCalls[0].Account.ID.Version())
	assert.Equal(t, uuid.Version(7), repo.createCalls[1].Account.ID.Version())
}

func TestAccountService_Open_NumberLayout(t *testing.T) {
	tests := []struct {
		name        string
		rawCurrency string
		wantCode    string
	}{
		{name: "default RUB", rawCurrency: "", wantCode: "810"},
		{name: "lower rub", rawCurrency: "rub", wantCode: "810"},
		{name: "upper USD", rawCurrency: "USD", wantCode: "840"},
		{name: "padded eur", rawCurrency: " eur ", wantCode: "978"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{nextSerial: 42, createResult: testAccount(TEST_NUMBER_CREATED, TEST_CURRENCY_RUB, models.STATUS_ACTIVE)}
			svc := newTestService(t, repo, &fakeCustomers{active: true}, &fakeLedger{}, TEST_CURRENCY_RUB)

			_, err := svc.Open(t.Context(), TEST_USER_ID, tt.rawCurrency)
			require.NoError(t, err)

			require.Len(t, repo.createCalls, 1)
			number := repo.createCalls[0].Account.Number
			require.Len(t, number, 20)
			assert.Equal(t, TEST_BALANCE_PREFIX, number[0:5])
			assert.Equal(t, tt.wantCode, number[5:8])
			assert.Equal(t, "00000000042", number[9:20])
		})
	}
}

func TestAccountService_Get(t *testing.T) {
	tests := []struct {
		name    string
		userID  *uuid.UUID
		repoErr error
		wantErr error
	}{
		{name: "with owner", userID: userPtr(TEST_USER_ID)},
		{name: "without owner", userID: nil},
		{name: "not found", userID: userPtr(TEST_USER_ID), repoErr: domain.ErrAccountNotFound, wantErr: domain.ErrAccountNotFound},
		{name: "db error", userID: nil, repoErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_ACTIVE)
			repo := &fakeAccountRepository{getByIDResult: account, getByIDErr: tt.repoErr}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			got, err := svc.Get(t.Context(), tt.userID, TEST_ACCOUNT_ID)

			assert.Equal(t, []getByIDCall{{UserID: tt.userID, AccountID: TEST_ACCOUNT_ID}}, repo.getByIDCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Same(t, account, got)
		})
	}
}

// pageCases - нормализация страницы по S12 (п.6 спецификации)
func pageCases() []struct {
	name string
	in   models.Page
	want models.Page
} {
	return []struct {
		name string
		in   models.Page
		want models.Page
	}{
		{name: "zero page", in: models.Page{Limit: 0, Offset: 0}, want: models.Page{Limit: 20, Offset: 0}},
		{name: "negative page", in: models.Page{Limit: -5, Offset: -3}, want: models.Page{Limit: 20, Offset: 0}},
		{name: "small page", in: models.Page{Limit: 1, Offset: 10}, want: models.Page{Limit: 1, Offset: 10}},
		{name: "max page", in: models.Page{Limit: 100, Offset: 0}, want: models.Page{Limit: 100, Offset: 0}},
		{name: "above max page", in: models.Page{Limit: 101, Offset: 0}, want: models.Page{Limit: 100, Offset: 0}},
		{name: "negative offset", in: models.Page{Limit: 50, Offset: -1}, want: models.Page{Limit: 50, Offset: 0}},
		{name: "default size as is", in: models.Page{Limit: 20, Offset: 5}, want: models.Page{Limit: 20, Offset: 5}},
		{name: "below max as is", in: models.Page{Limit: 99, Offset: 0}, want: models.Page{Limit: 99, Offset: 0}},
	}
}

func TestAccountService_List_Page(t *testing.T) {
	for _, tt := range pageCases() {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{listResult: []*models.Account{}}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			_, err := svc.List(t.Context(), TEST_USER_ID, tt.in)
			require.NoError(t, err)

			assert.Equal(t, []listCall{{UserID: TEST_USER_ID, Page: tt.want}}, repo.listCalls)
		})
	}
}

func TestAccountService_List(t *testing.T) {
	accounts := []*models.Account{
		testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_ACTIVE),
		testAccount(TEST_NUMBER_READ, TEST_CURRENCY_USD, models.STATUS_BLOCKED),
	}
	tests := []struct {
		name    string
		result  []*models.Account
		repoErr error
		wantErr error
	}{
		{name: "two accounts", result: accounts},
		{name: "empty list", result: []*models.Account{}},
		{name: "db error", repoErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{listResult: tt.result, listErr: tt.repoErr}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			got, err := svc.List(t.Context(), TEST_USER_ID, models.Page{Limit: 10, Offset: 0})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.result, got)
		})
	}
}

func TestAccountService_History_Page(t *testing.T) {
	for _, tt := range pageCases() {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{getByIDResult: testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_ACTIVE), historyResult: []*models.StatusChange{}}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			_, err := svc.History(t.Context(), TEST_ACCOUNT_ID, tt.in)
			require.NoError(t, err)

			assertAllGetByIDCalls(t, repo.getByIDCalls, getByIDCall{UserID: nil, AccountID: TEST_ACCOUNT_ID})
			assert.Equal(t, []historyCall{{AccountID: TEST_ACCOUNT_ID, Page: tt.want}}, repo.historyCalls)
		})
	}
}

func TestAccountService_History(t *testing.T) {
	moment := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	changes := []*models.StatusChange{
		{ID: 2, AccountID: TEST_ACCOUNT_ID, From: models.STATUS_ACTIVE, To: models.STATUS_BLOCKED, ActorID: userPtr(TEST_OFFICER_ID), Reason: TEST_REASON, CreatedAt: moment},
		{ID: 1, AccountID: TEST_ACCOUNT_ID, From: "", To: models.STATUS_ACTIVE, CreatedAt: moment},
	}
	page := models.Page{Limit: 10, Offset: 0}
	historyCalls := []historyCall{{AccountID: TEST_ACCOUNT_ID, Page: page}}
	tests := []struct {
		name             string
		getErr           error
		result           []*models.StatusChange
		historyErr       error
		wantHistoryCalls []historyCall
		wantErr          error
	}{
		{name: "two changes", result: changes, wantHistoryCalls: historyCalls},
		{name: "empty history", result: []*models.StatusChange{}, wantHistoryCalls: historyCalls},
		{name: "account not found", getErr: domain.ErrAccountNotFound, wantHistoryCalls: nil, wantErr: domain.ErrAccountNotFound},
		{name: "get db error", getErr: errDB, wantHistoryCalls: nil, wantErr: errDB},
		{name: "history db error", historyErr: errDB, wantHistoryCalls: historyCalls, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{
				getByIDResult: testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_ACTIVE),
				getByIDErr:    tt.getErr,
				historyResult: tt.result,
				historyErr:    tt.historyErr,
			}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			got, err := svc.History(t.Context(), TEST_ACCOUNT_ID, page)

			assertAllGetByIDCalls(t, repo.getByIDCalls, getByIDCall{UserID: nil, AccountID: TEST_ACCOUNT_ID})
			assert.Equal(t, tt.wantHistoryCalls, repo.historyCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.result, got)
		})
	}
}

// statusCase - общий кейс для Block и Unblock
type statusCase struct {
	name                string
	current             models.AccountStatus
	getErr              error
	transitionErr       error
	wantTransitionCalls []transitionCall
	wantNumber          string
	wantErr             error
}

func TestAccountService_Block(t *testing.T) {
	blockCalls := []transitionCall{{
		AccountID:   TEST_ACCOUNT_ID,
		Transition:  models.Transition{To: models.STATUS_BLOCKED, ActorID: userPtr(TEST_OFFICER_ID), Reason: TEST_REASON},
		AllowedFrom: []models.AccountStatus{models.STATUS_ACTIVE},
	}}
	tests := []statusCase{
		{name: "active to blocked", current: models.STATUS_ACTIVE, wantTransitionCalls: blockCalls, wantNumber: TEST_NUMBER_UPDATED},
		{name: "already blocked no-op", current: models.STATUS_BLOCKED, wantTransitionCalls: nil, wantNumber: TEST_NUMBER_READ},
		{name: "closed status conflict", current: models.STATUS_CLOSED, transitionErr: domain.ErrStatusConflict, wantTransitionCalls: blockCalls, wantErr: domain.ErrStatusConflict},
		{name: "get not found", getErr: domain.ErrAccountNotFound, wantTransitionCalls: nil, wantErr: domain.ErrAccountNotFound},
		{name: "get db error", getErr: errDB, wantTransitionCalls: nil, wantErr: errDB},
		{name: "transition not found", current: models.STATUS_ACTIVE, transitionErr: domain.ErrAccountNotFound, wantTransitionCalls: blockCalls, wantErr: domain.ErrAccountNotFound},
		{name: "transition db error", current: models.STATUS_ACTIVE, transitionErr: errDB, wantTransitionCalls: blockCalls, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{
				getByIDResult:    testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, tt.current),
				getByIDErr:       tt.getErr,
				transitionResult: testAccount(TEST_NUMBER_UPDATED, TEST_CURRENCY_RUB, models.STATUS_BLOCKED),
				transitionErr:    tt.transitionErr,
			}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			got, err := svc.Block(t.Context(), TEST_ACCOUNT_ID, TEST_OFFICER_ID, TEST_REASON)

			assertAllGetByIDCalls(t, repo.getByIDCalls, getByIDCall{UserID: nil, AccountID: TEST_ACCOUNT_ID})
			assert.Equal(t, tt.wantTransitionCalls, repo.transitionCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantNumber, got.Number)
		})
	}
}

func TestAccountService_Block_ReturnsTransitionResult(t *testing.T) {
	updated := testAccount(TEST_NUMBER_UPDATED, TEST_CURRENCY_RUB, models.STATUS_BLOCKED)
	repo := &fakeAccountRepository{getByIDResult: testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_ACTIVE), transitionResult: updated}
	svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

	got, err := svc.Block(t.Context(), TEST_ACCOUNT_ID, TEST_OFFICER_ID, TEST_REASON)
	require.NoError(t, err)

	assert.Same(t, updated, got)
}

func TestAccountService_Block_NoOpReturnsRead(t *testing.T) {
	read := testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_BLOCKED)
	repo := &fakeAccountRepository{getByIDResult: read}
	svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

	got, err := svc.Block(t.Context(), TEST_ACCOUNT_ID, TEST_OFFICER_ID, TEST_REASON)
	require.NoError(t, err)

	assert.Same(t, read, got)
}

func TestAccountService_Block_EmptyReason(t *testing.T) {
	repo := &fakeAccountRepository{
		getByIDResult:    testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, models.STATUS_ACTIVE),
		transitionResult: testAccount(TEST_NUMBER_UPDATED, TEST_CURRENCY_RUB, models.STATUS_BLOCKED),
	}
	svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

	got, err := svc.Block(t.Context(), TEST_ACCOUNT_ID, TEST_OFFICER_ID, "")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, TEST_NUMBER_UPDATED, got.Number)
	assert.Equal(t, []transitionCall{{
		AccountID:   TEST_ACCOUNT_ID,
		Transition:  models.Transition{To: models.STATUS_BLOCKED, ActorID: userPtr(TEST_OFFICER_ID), Reason: ""},
		AllowedFrom: []models.AccountStatus{models.STATUS_ACTIVE},
	}}, repo.transitionCalls)
}

func TestAccountService_Unblock(t *testing.T) {
	unblockCalls := []transitionCall{{
		AccountID:   TEST_ACCOUNT_ID,
		Transition:  models.Transition{To: models.STATUS_ACTIVE, ActorID: userPtr(TEST_OFFICER_ID), Reason: TEST_COMMENT},
		AllowedFrom: []models.AccountStatus{models.STATUS_BLOCKED},
	}}
	tests := []statusCase{
		{name: "blocked to active", current: models.STATUS_BLOCKED, wantTransitionCalls: unblockCalls, wantNumber: TEST_NUMBER_UPDATED},
		{name: "already active no-op", current: models.STATUS_ACTIVE, wantTransitionCalls: nil, wantNumber: TEST_NUMBER_READ},
		{name: "closed status conflict", current: models.STATUS_CLOSED, transitionErr: domain.ErrStatusConflict, wantTransitionCalls: unblockCalls, wantErr: domain.ErrStatusConflict},
		{name: "get not found", getErr: domain.ErrAccountNotFound, wantTransitionCalls: nil, wantErr: domain.ErrAccountNotFound},
		{name: "get db error", getErr: errDB, wantTransitionCalls: nil, wantErr: errDB},
		{name: "transition not found", current: models.STATUS_BLOCKED, transitionErr: domain.ErrAccountNotFound, wantTransitionCalls: unblockCalls, wantErr: domain.ErrAccountNotFound},
		{name: "transition db error", current: models.STATUS_BLOCKED, transitionErr: errDB, wantTransitionCalls: unblockCalls, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{
				getByIDResult:    testAccount(TEST_NUMBER_READ, TEST_CURRENCY_RUB, tt.current),
				getByIDErr:       tt.getErr,
				transitionResult: testAccount(TEST_NUMBER_UPDATED, TEST_CURRENCY_RUB, models.STATUS_ACTIVE),
				transitionErr:    tt.transitionErr,
			}
			svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

			got, err := svc.Unblock(t.Context(), TEST_ACCOUNT_ID, TEST_OFFICER_ID, TEST_COMMENT)

			assertAllGetByIDCalls(t, repo.getByIDCalls, getByIDCall{UserID: nil, AccountID: TEST_ACCOUNT_ID})
			assert.Equal(t, tt.wantTransitionCalls, repo.transitionCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantNumber, got.Number)
		})
	}
}

func TestAccountService_Close(t *testing.T) {
	closeCalls := []transitionCall{{
		AccountID:   TEST_ACCOUNT_ID,
		Transition:  models.Transition{To: models.STATUS_CLOSED, ActorID: nil, Reason: ""},
		AllowedFrom: []models.AccountStatus{models.STATUS_ACTIVE},
	}}
	usdLedgerCalls := []getBalanceCall{{AccountID: TEST_ACCOUNT_ID, Currency: TEST_CURRENCY_USD}}
	rubLedgerCalls := []getBalanceCall{{AccountID: TEST_ACCOUNT_ID, Currency: TEST_CURRENCY_RUB}}
	tests := []struct {
		name                string
		currency            models.Currency
		current             models.AccountStatus
		getErr              error
		amount              int64
		ledgerErr           error
		transitionErr       error
		wantLedgerCalls     []getBalanceCall
		wantTransitionCalls []transitionCall
		wantNumber          string
		wantErr             error
	}{
		{name: "active usd zero balance closes", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, amount: 0, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: closeCalls, wantNumber: TEST_NUMBER_UPDATED},
		{name: "active rub zero balance uses account currency", currency: TEST_CURRENCY_RUB, current: models.STATUS_ACTIVE, amount: 0, wantLedgerCalls: rubLedgerCalls, wantTransitionCalls: closeCalls, wantNumber: TEST_NUMBER_UPDATED},
		{name: "foreign or missing account", currency: TEST_CURRENCY_USD, getErr: domain.ErrAccountNotFound, wantLedgerCalls: nil, wantTransitionCalls: nil, wantErr: domain.ErrAccountNotFound},
		{name: "get db error", currency: TEST_CURRENCY_USD, getErr: errDB, wantLedgerCalls: nil, wantTransitionCalls: nil, wantErr: errDB},
		{name: "already closed no-op", currency: TEST_CURRENCY_USD, current: models.STATUS_CLOSED, wantLedgerCalls: nil, wantTransitionCalls: nil, wantNumber: TEST_NUMBER_READ},
		{name: "blocked status conflict", currency: TEST_CURRENCY_USD, current: models.STATUS_BLOCKED, wantLedgerCalls: nil, wantTransitionCalls: nil, wantErr: domain.ErrStatusConflict},
		{name: "ledger error", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, ledgerErr: errTransport, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: nil, wantErr: errTransport},
		{name: "positive balance", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, amount: 1, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: nil, wantErr: domain.ErrBalanceNotZero},
		{name: "negative balance", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, amount: -1, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: nil, wantErr: domain.ErrBalanceNotZero},
		{name: "transition status conflict", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, transitionErr: domain.ErrStatusConflict, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: closeCalls, wantErr: domain.ErrStatusConflict},
		{name: "transition not found", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, transitionErr: domain.ErrAccountNotFound, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: closeCalls, wantErr: domain.ErrAccountNotFound},
		{name: "transition db error", currency: TEST_CURRENCY_USD, current: models.STATUS_ACTIVE, transitionErr: errDB, wantLedgerCalls: usdLedgerCalls, wantTransitionCalls: closeCalls, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAccountRepository{
				getByIDResult:    testAccount(TEST_NUMBER_READ, tt.currency, tt.current),
				getByIDErr:       tt.getErr,
				transitionResult: testAccount(TEST_NUMBER_UPDATED, tt.currency, models.STATUS_CLOSED),
				transitionErr:    tt.transitionErr,
			}
			ledger := &fakeLedger{result: &models.Balance{AccountID: TEST_ACCOUNT_ID, Amount: tt.amount, Currency: tt.currency}, err: tt.ledgerErr}
			svc := newTestService(t, repo, &fakeCustomers{}, ledger, TEST_CURRENCY_RUB)

			got, err := svc.Close(t.Context(), TEST_USER_ID, TEST_ACCOUNT_ID)

			assertAllGetByIDCalls(t, repo.getByIDCalls, getByIDCall{UserID: userPtr(TEST_USER_ID), AccountID: TEST_ACCOUNT_ID})
			assert.Equal(t, tt.wantLedgerCalls, ledger.calls)
			assert.Equal(t, tt.wantTransitionCalls, repo.transitionCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantNumber, got.Number)
		})
	}
}

func TestAccountService_Close_ReturnsTransitionResult(t *testing.T) {
	updated := testAccount(TEST_NUMBER_UPDATED, TEST_CURRENCY_USD, models.STATUS_CLOSED)
	repo := &fakeAccountRepository{getByIDResult: testAccount(TEST_NUMBER_READ, TEST_CURRENCY_USD, models.STATUS_ACTIVE), transitionResult: updated}
	ledger := &fakeLedger{result: &models.Balance{AccountID: TEST_ACCOUNT_ID, Amount: 0, Currency: TEST_CURRENCY_USD}}
	svc := newTestService(t, repo, &fakeCustomers{}, ledger, TEST_CURRENCY_RUB)

	got, err := svc.Close(t.Context(), TEST_USER_ID, TEST_ACCOUNT_ID)
	require.NoError(t, err)

	assert.Same(t, updated, got)
}

func TestAccountService_Close_NoOpReturnsRead(t *testing.T) {
	read := testAccount(TEST_NUMBER_READ, TEST_CURRENCY_USD, models.STATUS_CLOSED)
	repo := &fakeAccountRepository{getByIDResult: read}
	svc := newTestService(t, repo, &fakeCustomers{}, &fakeLedger{}, TEST_CURRENCY_RUB)

	got, err := svc.Close(t.Context(), TEST_USER_ID, TEST_ACCOUNT_ID)
	require.NoError(t, err)

	assert.Same(t, read, got)
}

func TestAccountService_Balance(t *testing.T) {
	eurLedgerCalls := []getBalanceCall{{AccountID: TEST_ACCOUNT_ID, Currency: TEST_CURRENCY_EUR}}
	tests := []struct {
		name            string
		amount          int64
		getErr          error
		ledgerErr       error
		wantLedgerCalls []getBalanceCall
		wantErr         error
	}{
		{name: "eur balance", amount: 12345, wantLedgerCalls: eurLedgerCalls},
		{name: "zero balance", amount: 0, wantLedgerCalls: eurLedgerCalls},
		{name: "account not found", getErr: domain.ErrAccountNotFound, wantLedgerCalls: nil, wantErr: domain.ErrAccountNotFound},
		{name: "get db error", getErr: errDB, wantLedgerCalls: nil, wantErr: errDB},
		{name: "ledger error", ledgerErr: errTransport, wantLedgerCalls: eurLedgerCalls, wantErr: errTransport},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance := &models.Balance{AccountID: TEST_ACCOUNT_ID, Amount: tt.amount, Currency: TEST_CURRENCY_EUR}
			repo := &fakeAccountRepository{getByIDResult: testAccount(TEST_NUMBER_READ, TEST_CURRENCY_EUR, models.STATUS_ACTIVE), getByIDErr: tt.getErr}
			ledger := &fakeLedger{result: balance, err: tt.ledgerErr}
			svc := newTestService(t, repo, &fakeCustomers{}, ledger, TEST_CURRENCY_RUB)

			got, err := svc.Balance(t.Context(), TEST_USER_ID, TEST_ACCOUNT_ID)

			assertAllGetByIDCalls(t, repo.getByIDCalls, getByIDCall{UserID: userPtr(TEST_USER_ID), AccountID: TEST_ACCOUNT_ID})
			assert.Equal(t, tt.wantLedgerCalls, ledger.calls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Same(t, balance, got)
		})
	}
}
