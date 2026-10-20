package server

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
)

const (
	TEST_USER_ID        = "11111111-1111-4111-8111-111111111111"
	TEST_ACCOUNT_ID     = "22222222-2222-4222-8222-222222222222"
	TEST_ACCOUNT_ID_2   = "44444444-4444-4444-8444-444444444444"
	TEST_ACCOUNT_ID_3   = "55555555-5555-4555-8555-555555555555"
	TEST_OFFICER_ID     = "33333333-3333-4333-8333-333333333333"
	TEST_NIL_UUID       = "00000000-0000-0000-0000-000000000000"
	TEST_UPPER_UUID     = "AAAAAAAA-BBBB-4CCC-8DDD-EEEEEEEEEEEE"
	TEST_LOWER_UUID     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	TEST_BAD_UUID       = "abc"
	TEST_ACCOUNT_NUMBER = "40817810500000000001"
	TEST_CURRENCY       = "RUB"
)

// openCall - аргументы вызова Open.
type openCall struct {
	UserID   uuid.UUID
	Currency string
}

// getCall - аргументы вызова Get; UserID хранит копию значения по указателю.
type getCall struct {
	UserID    *uuid.UUID
	AccountID uuid.UUID
}

// listCall - аргументы вызова List.
type listCall struct {
	UserID uuid.UUID
	Page   models.Page
}

// blockCall - аргументы вызова Block.
type blockCall struct {
	AccountID uuid.UUID
	OfficerID uuid.UUID
	Reason    string
}

// unblockCall - аргументы вызова Unblock.
type unblockCall struct {
	AccountID uuid.UUID
	OfficerID uuid.UUID
	Comment   string
}

// closeCall - аргументы вызова Close.
type closeCall struct {
	UserID    uuid.UUID
	AccountID uuid.UUID
}

// balanceCall - аргументы вызова Balance.
type balanceCall struct {
	UserID    uuid.UUID
	AccountID uuid.UUID
}

// historyCall - аргументы вызова History.
type historyCall struct {
	AccountID uuid.UUID
	Page      models.Page
}

// fakeAccountService - ручной фейк сервиса счетов: отдаёт заданные значения и пишет вызовы.
type fakeAccountService struct {
	openAccount    *models.Account
	getAccount     *models.Account
	getByNumber    *models.Account
	listAccounts   []*models.Account
	blockAccount   *models.Account
	unblockAccount *models.Account
	closeAccount   *models.Account
	balance        *models.Balance
	history        []*models.StatusChange

	openErr    error
	getErr     error
	numberErr  error
	listErr    error
	blockErr   error
	unblockErr error
	closeErr   error
	balanceErr error
	historyErr error

	openCalls    []openCall
	getCalls     []getCall
	numberCalls  []string
	listCalls    []listCall
	blockCalls   []blockCall
	unblockCalls []unblockCall
	closeCalls   []closeCall
	balanceCalls []balanceCall
	historyCalls []historyCall
}

var _ interfaces.AccountService = (*fakeAccountService)(nil)

func (f *fakeAccountService) Open(_ context.Context, userID uuid.UUID, rawCurrency string) (*models.Account, error) {
	f.openCalls = append(f.openCalls, openCall{UserID: userID, Currency: rawCurrency})
	return f.openAccount, f.openErr
}

func (f *fakeAccountService) Get(_ context.Context, userID *uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	call := getCall{AccountID: accountID}
	if userID != nil {
		userIDCopy := *userID
		call.UserID = &userIDCopy
	}
	f.getCalls = append(f.getCalls, call)
	return f.getAccount, f.getErr
}

func (f *fakeAccountService) GetByNumber(_ context.Context, number string) (*models.Account, error) {
	f.numberCalls = append(f.numberCalls, number)
	return f.getByNumber, f.numberErr
}

func (f *fakeAccountService) List(_ context.Context, userID uuid.UUID, page models.Page) ([]*models.Account, error) {
	f.listCalls = append(f.listCalls, listCall{UserID: userID, Page: page})
	return f.listAccounts, f.listErr
}

func (f *fakeAccountService) Block(_ context.Context, accountID, officerID uuid.UUID, reason string) (*models.Account, error) {
	f.blockCalls = append(f.blockCalls, blockCall{AccountID: accountID, OfficerID: officerID, Reason: reason})
	return f.blockAccount, f.blockErr
}

func (f *fakeAccountService) Unblock(_ context.Context, accountID, officerID uuid.UUID, comment string) (*models.Account, error) {
	f.unblockCalls = append(f.unblockCalls, unblockCall{AccountID: accountID, OfficerID: officerID, Comment: comment})
	return f.unblockAccount, f.unblockErr
}

func (f *fakeAccountService) Close(_ context.Context, userID, accountID uuid.UUID) (*models.Account, error) {
	f.closeCalls = append(f.closeCalls, closeCall{UserID: userID, AccountID: accountID})
	return f.closeAccount, f.closeErr
}

func (f *fakeAccountService) Balance(_ context.Context, userID, accountID uuid.UUID) (*models.Balance, error) {
	f.balanceCalls = append(f.balanceCalls, balanceCall{UserID: userID, AccountID: accountID})
	return f.balance, f.balanceErr
}

func (f *fakeAccountService) History(_ context.Context, accountID uuid.UUID, page models.Page) ([]*models.StatusChange, error) {
	f.historyCalls = append(f.historyCalls, historyCall{AccountID: accountID, Page: page})
	return f.history, f.historyErr
}

// totalCalls - число вызовов всех методов фейка.
func (f *fakeAccountService) totalCalls() int {
	return len(f.openCalls) + len(f.getCalls) + len(f.numberCalls) + len(f.listCalls) + len(f.blockCalls) +
		len(f.unblockCalls) + len(f.closeCalls) + len(f.balanceCalls) + len(f.historyCalls)
}

// newTestServer собирает сервер поверх фейка.
func newTestServer(fake *fakeAccountService) *AccountServer {
	return NewAccountServer(&service.Service{Account: fake}, slog.New(slog.DiscardHandler))
}

// requireProtoEqual сравнивает proto-сообщения и печатает оба при расхождении.
func requireProtoEqual(t *testing.T, want, got proto.Message) {
	t.Helper()
	if !proto.Equal(want, got) {
		t.Fatalf("proto messages differ:\nwant: %v\ngot:  %v", want, got)
	}
}

// testStatusChangedAt - фиксированное время смены статуса.
func testStatusChangedAt() time.Time {
	return time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
}

// testCreatedAt - фиксированное время создания из примера п.6.
func testCreatedAt() time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}

// testUpdatedAt - фиксированное время обновления с наносекундами.
func testUpdatedAt() time.Time {
	return time.Date(2026, 3, 4, 5, 6, 7, 123456789, time.UTC)
}

// newAccountModel - доменный счёт пользователя TEST_USER_ID.
func newAccountModel(accountID string, accountStatus models.AccountStatus) *models.Account {
	return &models.Account{
		ID:              uuid.MustParse(accountID),
		UserID:          uuid.MustParse(TEST_USER_ID),
		Number:          TEST_ACCOUNT_NUMBER,
		Currency:        models.Currency(TEST_CURRENCY),
		Status:          accountStatus,
		StatusChangedAt: testStatusChangedAt(),
		CreatedAt:       testCreatedAt(),
		UpdatedAt:       testUpdatedAt(),
	}
}

// newAccountProto - ожидаемый proto для newAccountModel.
func newAccountProto(accountID string, accountStatus accountv1.AccountStatus) *accountv1.Account {
	return &accountv1.Account{
		Id:              accountID,
		UserId:          TEST_USER_ID,
		Number:          TEST_ACCOUNT_NUMBER,
		Currency:        TEST_CURRENCY,
		Status:          accountStatus,
		StatusChangedAt: timestamppb.New(testStatusChangedAt()),
		CreatedAt:       timestamppb.New(testCreatedAt()),
		UpdatedAt:       timestamppb.New(testUpdatedAt()),
	}
}

// rpcCase - вызов одного RPC с валидным запросом и способ задать ошибку его метода сервиса.
type rpcCase struct {
	name      string
	call      func(t *testing.T, s *AccountServer) (bool, error)
	setErr    func(f *fakeAccountService, err error)
	ownCalls  func(f *fakeAccountService) int
	stubError error
}

// allRPCs - все 9 RPC с валидными запросами; первый результат call - ответ nil.
func allRPCs() []rpcCase {
	return []rpcCase{
		{
			name: "OpenAccount",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.OpenAccount(t.Context(), &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: TEST_CURRENCY})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.openErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.openCalls) },
			stubError: domain.ErrOpenFailed,
		},
		{
			name: "GetAccount",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.getErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.getCalls) },
			stubError: domain.ErrGetFailed,
		},
		{
			name: "GetAccountByNumber",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccountByNumber(t.Context(), &accountv1.GetAccountByNumberRequest{Number: TEST_ACCOUNT_NUMBER})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.numberErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.numberCalls) },
			stubError: domain.ErrGetFailed,
		},
		{
			name: "ListAccounts",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.ListAccounts(t.Context(), &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 0})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.listErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.listCalls) },
			stubError: domain.ErrListFailed,
		},
		{
			name: "BlockAccount",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.BlockAccount(t.Context(), &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "fraud"})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.blockErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.blockCalls) },
			stubError: domain.ErrUpdateFailed,
		},
		{
			name: "UnblockAccount",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.UnblockAccount(t.Context(), &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "checked"})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.unblockErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.unblockCalls) },
			stubError: domain.ErrUpdateFailed,
		},
		{
			name: "CloseAccount",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.CloseAccount(t.Context(), &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.closeErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.closeCalls) },
			stubError: domain.ErrUpdateFailed,
		},
		{
			name: "GetBalance",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetBalance(t.Context(), &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.balanceErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.balanceCalls) },
			stubError: domain.ErrBalanceFailed,
		},
		{
			name: "ListStatusHistory",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.ListStatusHistory(t.Context(), &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 20, Offset: 0})
				return resp == nil, err
			},
			setErr:    func(f *fakeAccountService, err error) { f.historyErr = err },
			ownCalls:  func(f *fakeAccountService) int { return len(f.historyCalls) },
			stubError: domain.ErrHistoryFailed,
		},
	}
}
