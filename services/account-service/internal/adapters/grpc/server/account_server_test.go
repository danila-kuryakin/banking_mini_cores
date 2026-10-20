package server

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
)

// uuidPtr - указатель на разобранный UUID.
func uuidPtr(raw string) *uuid.UUID {
	parsed := uuid.MustParse(raw)
	return &parsed
}

// TestAccountServer_ParseErrors проверяет разбор UUID по порядку полей proto и отсутствие вызовов сервиса.
func TestAccountServer_ParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		call    func(t *testing.T, s *AccountServer) (bool, error)
		wantErr error
	}{
		{
			name: "OpenAccount bad user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.OpenAccount(t.Context(), &accountv1.OpenAccountRequest{UserId: TEST_BAD_UUID, Currency: TEST_CURRENCY})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "OpenAccount empty user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.OpenAccount(t.Context(), &accountv1.OpenAccountRequest{UserId: "", Currency: TEST_CURRENCY})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "ListAccounts bad user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.ListAccounts(t.Context(), &accountv1.ListAccountsRequest{UserId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "ListAccounts empty user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.ListAccounts(t.Context(), &accountv1.ListAccountsRequest{UserId: ""})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "ListStatusHistory bad account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.ListStatusHistory(t.Context(), &accountv1.ListStatusHistoryRequest{AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "ListStatusHistory empty account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.ListStatusHistory(t.Context(), &accountv1.ListStatusHistoryRequest{AccountId: ""})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "BlockAccount bad account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.BlockAccount(t.Context(), &accountv1.BlockAccountRequest{AccountId: TEST_BAD_UUID, OfficerId: TEST_OFFICER_ID, Reason: "fraud"})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "BlockAccount bad officer_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.BlockAccount(t.Context(), &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_BAD_UUID, Reason: "fraud"})
				return resp == nil, err
			},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name: "BlockAccount both bad - account_id first",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.BlockAccount(t.Context(), &accountv1.BlockAccountRequest{AccountId: TEST_BAD_UUID, OfficerId: TEST_BAD_UUID, Reason: "fraud"})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "BlockAccount empty officer_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.BlockAccount(t.Context(), &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: "", Reason: "fraud"})
				return resp == nil, err
			},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name: "UnblockAccount bad account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.UnblockAccount(t.Context(), &accountv1.UnblockAccountRequest{AccountId: TEST_BAD_UUID, OfficerId: TEST_OFFICER_ID, Comment: "ok"})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "UnblockAccount bad officer_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.UnblockAccount(t.Context(), &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_BAD_UUID, Comment: "ok"})
				return resp == nil, err
			},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name: "UnblockAccount both bad - account_id first",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.UnblockAccount(t.Context(), &accountv1.UnblockAccountRequest{AccountId: TEST_BAD_UUID, OfficerId: TEST_BAD_UUID, Comment: "ok"})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "UnblockAccount empty officer_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.UnblockAccount(t.Context(), &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: "", Comment: "ok"})
				return resp == nil, err
			},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{
			name: "CloseAccount bad user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.CloseAccount(t.Context(), &accountv1.CloseAccountRequest{UserId: TEST_BAD_UUID, AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "CloseAccount bad account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.CloseAccount(t.Context(), &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "CloseAccount both bad - user_id first",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.CloseAccount(t.Context(), &accountv1.CloseAccountRequest{UserId: TEST_BAD_UUID, AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "CloseAccount empty user_id is required",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.CloseAccount(t.Context(), &accountv1.CloseAccountRequest{UserId: "", AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetBalance bad user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetBalance(t.Context(), &accountv1.GetBalanceRequest{UserId: TEST_BAD_UUID, AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetBalance bad account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetBalance(t.Context(), &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "GetBalance both bad - user_id first",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetBalance(t.Context(), &accountv1.GetBalanceRequest{UserId: TEST_BAD_UUID, AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetBalance empty user_id is required",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetBalance(t.Context(), &accountv1.GetBalanceRequest{UserId: "", AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetAccount space user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: " ", AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetAccount bad user_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: TEST_BAD_UUID, AccountId: TEST_ACCOUNT_ID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetAccount empty user_id and bad account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: "", AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "GetAccount both bad - user_id first",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: TEST_BAD_UUID, AccountId: TEST_BAD_UUID})
				return resp == nil, err
			},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name: "GetAccount empty account_id",
			call: func(t *testing.T, s *AccountServer) (bool, error) {
				resp, err := s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: ""})
				return resp == nil, err
			},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			srv := newTestServer(fake)

			respIsNil, err := tt.call(t, srv)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.True(t, respIsNil)
			assert.Equal(t, 0, fake.totalCalls())
		})
	}
}

// TestAccountServer_OpenAccount проверяет аргументы Open и маппинг ответа.
func TestAccountServer_OpenAccount(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.OpenAccountRequest
		wantCall []openCall
	}{
		{
			name:     "typical",
			req:      &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: TEST_CURRENCY},
			wantCall: []openCall{{UserID: uuid.MustParse(TEST_USER_ID), Currency: TEST_CURRENCY}},
		},
		{
			name:     "currency passed as is",
			req:      &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: " usd "},
			wantCall: []openCall{{UserID: uuid.MustParse(TEST_USER_ID), Currency: " usd "}},
		},
		{
			name:     "nil uuid user_id is valid",
			req:      &accountv1.OpenAccountRequest{UserId: TEST_NIL_UUID, Currency: TEST_CURRENCY},
			wantCall: []openCall{{UserID: uuid.Nil, Currency: TEST_CURRENCY}},
		},
		{
			name:     "upper case user_id is valid",
			req:      &accountv1.OpenAccountRequest{UserId: TEST_UPPER_UUID, Currency: TEST_CURRENCY},
			wantCall: []openCall{{UserID: uuid.MustParse(TEST_LOWER_UUID), Currency: TEST_CURRENCY}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{openAccount: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE)}
			srv := newTestServer(fake)

			resp, err := srv.OpenAccount(t.Context(), tt.req)

			require.NoError(t, err)
			requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE), resp)
			assert.Equal(t, tt.wantCall, fake.openCalls)
		})
	}
}

// TestAccountServer_GetAccount проверяет необязательного владельца и аргументы Get.
func TestAccountServer_GetAccount(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.GetAccountRequest
		wantCall []getCall
	}{
		{
			name:     "empty user_id gives nil owner",
			req:      &accountv1.GetAccountRequest{UserId: "", AccountId: TEST_ACCOUNT_ID},
			wantCall: []getCall{{UserID: nil, AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
		},
		{
			name:     "user_id passed by pointer",
			req:      &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			wantCall: []getCall{{UserID: uuidPtr(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
		},
		{
			name:     "upper case user_id is valid",
			req:      &accountv1.GetAccountRequest{UserId: TEST_UPPER_UUID, AccountId: TEST_ACCOUNT_ID},
			wantCall: []getCall{{UserID: uuidPtr(TEST_LOWER_UUID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
		},
		{
			name:     "nil uuid user_id is not empty",
			req:      &accountv1.GetAccountRequest{UserId: TEST_NIL_UUID, AccountId: TEST_NIL_UUID},
			wantCall: []getCall{{UserID: uuidPtr(TEST_NIL_UUID), AccountID: uuid.Nil}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{getAccount: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE)}
			srv := newTestServer(fake)

			resp, err := srv.GetAccount(t.Context(), tt.req)

			require.NoError(t, err)
			requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE), resp)
			assert.Equal(t, tt.wantCall, fake.getCalls)
		})
	}
}

// TestAccountServer_GetAccountByNumber_InvalidNumber проверяет разбор номера: не 20 цифр - InvalidArgument без вызова сервиса.
func TestAccountServer_GetAccountByNumber_InvalidNumber(t *testing.T) {
	tests := []struct {
		name   string
		number string
	}{
		{name: "empty", number: ""},
		{name: "19 digits", number: "4081781050000000000"},
		{name: "21 digits", number: "408178105000000000011"},
		{name: "letters", number: "40817810A00000000001"},
		{name: "spaces around", number: " 40817810500000000001"},
		{name: "minus sign", number: "-4081781050000000001"},
		{name: "non-ascii digits", number: "40817810500000000٠٠1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			srv := newTestServer(fake)

			resp, err := srv.GetAccountByNumber(t.Context(), &accountv1.GetAccountByNumberRequest{Number: tt.number})

			require.ErrorIs(t, err, domain.ErrNumberIsInvalid)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, resp)
			assert.Zero(t, fake.totalCalls())
		})
	}
}

// TestAccountServer_GetAccountByNumber проверяет успех и передачу номера в сервис как есть.
func TestAccountServer_GetAccountByNumber(t *testing.T) {
	fake := &fakeAccountService{getByNumber: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_BLOCKED)}
	srv := newTestServer(fake)

	resp, err := srv.GetAccountByNumber(t.Context(), &accountv1.GetAccountByNumberRequest{Number: TEST_ACCOUNT_NUMBER})

	require.NoError(t, err)
	requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED), resp)
	assert.Equal(t, []string{TEST_ACCOUNT_NUMBER}, fake.numberCalls)
}

// TestAccountServer_GetAccountByNumber_NotFound проверяет NOT_FOUND "account not found" для неизвестного номера.
func TestAccountServer_GetAccountByNumber_NotFound(t *testing.T) {
	fake := &fakeAccountService{numberErr: fmt.Errorf("get: %w", domain.ErrAccountNotFound)}
	srv := newTestServer(fake)

	resp, err := srv.GetAccountByNumber(t.Context(), &accountv1.GetAccountByNumberRequest{Number: TEST_ACCOUNT_NUMBER})

	require.ErrorIs(t, err, domain.ErrAccountMissing)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, "account not found", status.Convert(err).Message())
	assert.Nil(t, resp)
}

// TestAccountServer_AccountMapping проверяет маппинг статуса и времени счёта.
func TestAccountServer_AccountMapping(t *testing.T) {
	zeroTimeAccount := newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE)
	zeroTimeAccount.StatusChangedAt = time.Time{}
	zeroTimeAccount.CreatedAt = time.Time{}
	zeroTimeAccount.UpdatedAt = time.Time{}
	zeroTimeProto := newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE)
	zeroTimeProto.StatusChangedAt = timestamppb.New(time.Time{})
	zeroTimeProto.CreatedAt = timestamppb.New(time.Time{})
	zeroTimeProto.UpdatedAt = timestamppb.New(time.Time{})

	tests := []struct {
		name  string
		model *models.Account
		want  *accountv1.Account
	}{
		{
			name:  "active",
			model: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
			want:  newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE),
		},
		{
			name:  "blocked",
			model: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_BLOCKED),
			want:  newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED),
		},
		{
			name:  "closed",
			model: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_CLOSED),
			want:  newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED),
		},
		{
			name:  "empty status",
			model: newAccountModel(TEST_ACCOUNT_ID, ""),
			want:  newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED),
		},
		{
			name:  "unknown status",
			model: newAccountModel(TEST_ACCOUNT_ID, "frozen"),
			want:  newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED),
		},
		{
			name:  "upper case status is unknown",
			model: newAccountModel(TEST_ACCOUNT_ID, "ACTIVE"),
			want:  newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED),
		},
		{
			name:  "zero times",
			model: zeroTimeAccount,
			want:  zeroTimeProto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{getAccount: tt.model}
			srv := newTestServer(fake)

			resp, err := srv.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID})

			require.NoError(t, err)
			requireProtoEqual(t, tt.want, resp)
		})
	}
}

// TestAccountServer_AccountMappingInvariant проверяет одинаковый маппинг счёта во всех RPC, отдающих Account.
func TestAccountServer_AccountMappingInvariant(t *testing.T) {
	tests := []struct {
		name string
		call func(t *testing.T, s *AccountServer, f *fakeAccountService, model *models.Account) (*accountv1.Account, error)
	}{
		{
			name: "OpenAccount",
			call: func(t *testing.T, s *AccountServer, f *fakeAccountService, model *models.Account) (*accountv1.Account, error) {
				f.openAccount = model
				return s.OpenAccount(t.Context(), &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: TEST_CURRENCY})
			},
		},
		{
			name: "GetAccount",
			call: func(t *testing.T, s *AccountServer, f *fakeAccountService, model *models.Account) (*accountv1.Account, error) {
				f.getAccount = model
				return s.GetAccount(t.Context(), &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID})
			},
		},
		{
			name: "BlockAccount",
			call: func(t *testing.T, s *AccountServer, f *fakeAccountService, model *models.Account) (*accountv1.Account, error) {
				f.blockAccount = model
				return s.BlockAccount(t.Context(), &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "fraud"})
			},
		},
		{
			name: "UnblockAccount",
			call: func(t *testing.T, s *AccountServer, f *fakeAccountService, model *models.Account) (*accountv1.Account, error) {
				f.unblockAccount = model
				return s.UnblockAccount(t.Context(), &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "ok"})
			},
		},
		{
			name: "CloseAccount",
			call: func(t *testing.T, s *AccountServer, f *fakeAccountService, model *models.Account) (*accountv1.Account, error) {
				f.closeAccount = model
				return s.CloseAccount(t.Context(), &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			srv := newTestServer(fake)

			resp, err := tt.call(t, srv, fake, newAccountModel(TEST_ACCOUNT_ID, models.STATUS_BLOCKED))

			require.NoError(t, err)
			requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED), resp)
		})
	}
}

// TestAccountServer_ListAccounts проверяет страницу без нормализации, порядок и пустой список.
func TestAccountServer_ListAccounts(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.ListAccountsRequest
		accounts []*models.Account
		wantCall []listCall
		want     []*accountv1.Account
	}{
		{
			name: "zero page",
			req:  &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 0, Offset: 0},
			accounts: []*models.Account{
				newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
			},
			wantCall: []listCall{{UserID: uuid.MustParse(TEST_USER_ID), Page: models.Page{Limit: 0, Offset: 0}}},
			want: []*accountv1.Account{
				newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE),
			},
		},
		{
			name: "page above ceiling is not normalized",
			req:  &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 150, Offset: 7},
			accounts: []*models.Account{
				newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
				newAccountModel(TEST_ACCOUNT_ID_2, models.STATUS_BLOCKED),
				newAccountModel(TEST_ACCOUNT_ID_3, models.STATUS_CLOSED),
			},
			wantCall: []listCall{{UserID: uuid.MustParse(TEST_USER_ID), Page: models.Page{Limit: 150, Offset: 7}}},
			want: []*accountv1.Account{
				newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE),
				newAccountProto(TEST_ACCOUNT_ID_2, accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED),
				newAccountProto(TEST_ACCOUNT_ID_3, accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED),
			},
		},
		{
			name: "order from service is kept",
			req:  &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 0},
			accounts: []*models.Account{
				newAccountModel(TEST_ACCOUNT_ID_3, models.STATUS_CLOSED),
				newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
			},
			wantCall: []listCall{{UserID: uuid.MustParse(TEST_USER_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want: []*accountv1.Account{
				newAccountProto(TEST_ACCOUNT_ID_3, accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED),
				newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE),
			},
		},
		{
			name:     "negative page is not normalized",
			req:      &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: -1, Offset: -5},
			accounts: []*models.Account{},
			wantCall: []listCall{{UserID: uuid.MustParse(TEST_USER_ID), Page: models.Page{Limit: -1, Offset: -5}}},
			want:     nil,
		},
		{
			name:     "empty slice from service",
			req:      &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 0},
			accounts: []*models.Account{},
			wantCall: []listCall{{UserID: uuid.MustParse(TEST_USER_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want:     nil,
		},
		{
			name:     "nil slice from service",
			req:      &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 0},
			accounts: nil,
			wantCall: []listCall{{UserID: uuid.MustParse(TEST_USER_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{listAccounts: tt.accounts}
			srv := newTestServer(fake)

			resp, err := srv.ListAccounts(t.Context(), tt.req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			requireProtoEqual(t, &accountv1.ListAccountsResponse{Accounts: tt.want}, resp)
			assert.Len(t, resp.GetAccounts(), len(tt.want))
			assert.Equal(t, tt.wantCall, fake.listCalls)
		})
	}
}

// TestAccountServer_BlockAccount проверяет аргументы Block и обрезку причины.
func TestAccountServer_BlockAccount(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.BlockAccountRequest
		wantCall []blockCall
	}{
		{
			name:     "reason is trimmed",
			req:      &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "  fraud  "},
			wantCall: []blockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Reason: "fraud"}},
		},
		{
			name:     "reason of spaces becomes empty",
			req:      &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "   "},
			wantCall: []blockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Reason: ""}},
		},
		{
			name:     "empty reason",
			req:      &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: ""},
			wantCall: []blockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Reason: ""}},
		},
		{
			name:     "tab and newline are trimmed",
			req:      &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "\tfraud\n"},
			wantCall: []blockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Reason: "fraud"}},
		},
		{
			name:     "inner spaces are kept",
			req:      &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "a  b"},
			wantCall: []blockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Reason: "a  b"}},
		},
		{
			name:     "nil uuid officer_id is valid",
			req:      &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_NIL_UUID, Reason: "fraud"},
			wantCall: []blockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.Nil, Reason: "fraud"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{blockAccount: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_BLOCKED)}
			srv := newTestServer(fake)

			resp, err := srv.BlockAccount(t.Context(), tt.req)

			require.NoError(t, err)
			requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED), resp)
			assert.Equal(t, tt.wantCall, fake.blockCalls)
		})
	}
}

// TestAccountServer_UnblockAccount проверяет аргументы Unblock и обрезку комментария.
func TestAccountServer_UnblockAccount(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.UnblockAccountRequest
		wantCall []unblockCall
	}{
		{
			name:     "comment is trimmed",
			req:      &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "  checked  "},
			wantCall: []unblockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Comment: "checked"}},
		},
		{
			name:     "comment of spaces becomes empty",
			req:      &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "   "},
			wantCall: []unblockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Comment: ""}},
		},
		{
			name:     "empty comment",
			req:      &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: ""},
			wantCall: []unblockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Comment: ""}},
		},
		{
			name:     "tab and newline are trimmed",
			req:      &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "\tchecked\n"},
			wantCall: []unblockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Comment: "checked"}},
		},
		{
			name:     "inner spaces are kept",
			req:      &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "a  b"},
			wantCall: []unblockCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID), Comment: "a  b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{unblockAccount: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE)}
			srv := newTestServer(fake)

			resp, err := srv.UnblockAccount(t.Context(), tt.req)

			require.NoError(t, err)
			requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE), resp)
			assert.Equal(t, tt.wantCall, fake.unblockCalls)
		})
	}
}

// TestAccountServer_CloseAccount проверяет порядок аргументов Close: пользователь, потом счёт.
func TestAccountServer_CloseAccount(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.CloseAccountRequest
		wantCall []closeCall
	}{
		{
			name:     "typical",
			req:      &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			wantCall: []closeCall{{UserID: uuid.MustParse(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
		},
		{
			name:     "nil uuid ids are valid",
			req:      &accountv1.CloseAccountRequest{UserId: TEST_NIL_UUID, AccountId: TEST_NIL_UUID},
			wantCall: []closeCall{{UserID: uuid.Nil, AccountID: uuid.Nil}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{closeAccount: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_CLOSED)}
			srv := newTestServer(fake)

			resp, err := srv.CloseAccount(t.Context(), tt.req)

			require.NoError(t, err)
			requireProtoEqual(t, newAccountProto(TEST_ACCOUNT_ID, accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED), resp)
			assert.Equal(t, tt.wantCall, fake.closeCalls)
		})
	}
}

// TestAccountServer_GetBalance проверяет аргументы Balance и сборку ответа.
func TestAccountServer_GetBalance(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.GetBalanceRequest
		balance  *models.Balance
		wantCall []balanceCall
		want     *accountv1.GetBalanceResponse
	}{
		{
			name:     "typical",
			req:      &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			balance:  &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 12345, Currency: "USD"},
			wantCall: []balanceCall{{UserID: uuid.MustParse(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
			want:     &accountv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: 12345, Currency: "USD"},
		},
		{
			name:     "zero amount",
			req:      &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			balance:  &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_CURRENCY},
			wantCall: []balanceCall{{UserID: uuid.MustParse(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
			want:     &accountv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: 0, Currency: TEST_CURRENCY},
		},
		{
			name:     "negative amount",
			req:      &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			balance:  &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: -500, Currency: TEST_CURRENCY},
			wantCall: []balanceCall{{UserID: uuid.MustParse(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
			want:     &accountv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: -500, Currency: TEST_CURRENCY},
		},
		{
			name:     "max int64 amount",
			req:      &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			balance:  &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: math.MaxInt64, Currency: TEST_CURRENCY},
			wantCall: []balanceCall{{UserID: uuid.MustParse(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
			want:     &accountv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID, Amount: math.MaxInt64, Currency: TEST_CURRENCY},
		},
		{
			name:     "account_id taken from balance",
			req:      &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID},
			balance:  &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID_2), Amount: 1, Currency: TEST_CURRENCY},
			wantCall: []balanceCall{{UserID: uuid.MustParse(TEST_USER_ID), AccountID: uuid.MustParse(TEST_ACCOUNT_ID)}},
			want:     &accountv1.GetBalanceResponse{AccountId: TEST_ACCOUNT_ID_2, Amount: 1, Currency: TEST_CURRENCY},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{balance: tt.balance}
			srv := newTestServer(fake)

			resp, err := srv.GetBalance(t.Context(), tt.req)

			require.NoError(t, err)
			requireProtoEqual(t, tt.want, resp)
			assert.Equal(t, tt.wantCall, fake.balanceCalls)
		})
	}
}

// TestAccountServer_ListStatusHistory проверяет страницу, маппинг записей и порядок.
func TestAccountServer_ListStatusHistory(t *testing.T) {
	tests := []struct {
		name     string
		req      *accountv1.ListStatusHistoryRequest
		history  []*models.StatusChange
		wantCall []historyCall
		want     []*accountv1.StatusChange
	}{
		{
			name: "two changes in service order",
			req:  &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 0, Offset: 0},
			history: []*models.StatusChange{
				{ID: 2, AccountID: uuid.MustParse(TEST_ACCOUNT_ID), From: models.STATUS_ACTIVE, To: models.STATUS_BLOCKED, ActorID: uuidPtr(TEST_OFFICER_ID), Reason: "fraud", CreatedAt: testUpdatedAt()},
				{ID: 1, AccountID: uuid.MustParse(TEST_ACCOUNT_ID), From: "", To: models.STATUS_ACTIVE, ActorID: nil, Reason: "", CreatedAt: testCreatedAt()},
			},
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: 0, Offset: 0}}},
			want: []*accountv1.StatusChange{
				{Id: 2, AccountId: TEST_ACCOUNT_ID, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED, ActorId: TEST_OFFICER_ID, Reason: "fraud", CreatedAt: timestamppb.New(testUpdatedAt())},
				{Id: 1, AccountId: TEST_ACCOUNT_ID, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE, ActorId: "", Reason: "", CreatedAt: timestamppb.New(testCreatedAt())},
			},
		},
		{
			name: "closed blocked and unknown statuses",
			req:  &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 150, Offset: 7},
			history: []*models.StatusChange{
				{ID: 3, AccountID: uuid.MustParse(TEST_ACCOUNT_ID), From: models.STATUS_BLOCKED, To: models.STATUS_CLOSED, ActorID: uuidPtr(TEST_OFFICER_ID), Reason: "r", CreatedAt: testCreatedAt()},
				{ID: 4, AccountID: uuid.MustParse(TEST_ACCOUNT_ID), From: "frozen", To: "", ActorID: nil, Reason: "", CreatedAt: testCreatedAt()},
			},
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: 150, Offset: 7}}},
			want: []*accountv1.StatusChange{
				{Id: 3, AccountId: TEST_ACCOUNT_ID, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED, ActorId: TEST_OFFICER_ID, Reason: "r", CreatedAt: timestamppb.New(testCreatedAt())},
				{Id: 4, AccountId: TEST_ACCOUNT_ID, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED, ActorId: "", Reason: "", CreatedAt: timestamppb.New(testCreatedAt())},
			},
		},
		{
			name: "nil uuid actor is not empty",
			req:  &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 20, Offset: 0},
			history: []*models.StatusChange{
				{ID: 5, AccountID: uuid.MustParse(TEST_ACCOUNT_ID), From: models.STATUS_ACTIVE, To: models.STATUS_BLOCKED, ActorID: uuidPtr(TEST_NIL_UUID), Reason: "x", CreatedAt: testCreatedAt()},
			},
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want: []*accountv1.StatusChange{
				{Id: 5, AccountId: TEST_ACCOUNT_ID, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED, ActorId: TEST_NIL_UUID, Reason: "x", CreatedAt: timestamppb.New(testCreatedAt())},
			},
		},
		{
			name: "zero time zero id and max id",
			req:  &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 20, Offset: 0},
			history: []*models.StatusChange{
				{ID: 0, AccountID: uuid.MustParse(TEST_ACCOUNT_ID), From: "", To: models.STATUS_ACTIVE, CreatedAt: time.Time{}},
				{ID: math.MaxInt64, AccountID: uuid.MustParse(TEST_ACCOUNT_ID_2), From: models.STATUS_ACTIVE, To: models.STATUS_CLOSED, CreatedAt: time.Time{}},
			},
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want: []*accountv1.StatusChange{
				{Id: 0, AccountId: TEST_ACCOUNT_ID, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE, CreatedAt: timestamppb.New(time.Time{})},
				{Id: math.MaxInt64, AccountId: TEST_ACCOUNT_ID_2, FromStatus: accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE, ToStatus: accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED, CreatedAt: timestamppb.New(time.Time{})},
			},
		},
		{
			name:     "negative page is not normalized",
			req:      &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: -1, Offset: -5},
			history:  []*models.StatusChange{},
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: -1, Offset: -5}}},
			want:     nil,
		},
		{
			name:     "empty slice from service",
			req:      &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 20, Offset: 0},
			history:  []*models.StatusChange{},
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want:     nil,
		},
		{
			name:     "nil slice from service",
			req:      &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 20, Offset: 0},
			history:  nil,
			wantCall: []historyCall{{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Page: models.Page{Limit: 20, Offset: 0}}},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{history: tt.history}
			srv := newTestServer(fake)

			resp, err := srv.ListStatusHistory(t.Context(), tt.req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			requireProtoEqual(t, &accountv1.ListStatusHistoryResponse{Changes: tt.want}, resp)
			assert.Len(t, resp.GetChanges(), len(tt.want))
			assert.Equal(t, tt.wantCall, fake.historyCalls)
		})
	}
}

// TestAccountServer_SentinelErrors проверяет перевод внутренних sentinel в gRPC-статусы во всех RPC.
func TestAccountServer_SentinelErrors(t *testing.T) {
	sentinels := []struct {
		name     string
		internal error
		want     error
		wantCode codes.Code
	}{
		{name: "account not found", internal: domain.ErrAccountNotFound, want: domain.ErrAccountMissing, wantCode: codes.NotFound},
		{name: "customer not found", internal: domain.ErrCustomerNotFound, want: domain.ErrCustomerMissing, wantCode: codes.NotFound},
		{name: "customer not active", internal: domain.ErrCustomerNotActive, want: domain.ErrCustomerIsNotActive, wantCode: codes.FailedPrecondition},
		{name: "account limit", internal: domain.ErrAccountLimitReached, want: domain.ErrAccountLimit, wantCode: codes.FailedPrecondition},
		{name: "status conflict", internal: domain.ErrStatusConflict, want: domain.ErrTransitionForbidden, wantCode: codes.FailedPrecondition},
		{name: "balance not zero", internal: domain.ErrBalanceNotZero, want: domain.ErrBalanceIsNotZero, wantCode: codes.FailedPrecondition},
	}

	type errorCase struct {
		name     string
		rpc      rpcCase
		svcErr   error
		want     error
		wantCode codes.Code
	}
	var tests []errorCase
	for _, rpc := range allRPCs() {
		for _, s := range sentinels {
			tests = append(tests,
				errorCase{name: rpc.name + "/" + s.name + "/bare", rpc: rpc, svcErr: s.internal, want: s.want, wantCode: s.wantCode},
				errorCase{name: rpc.name + "/" + s.name + "/wrapped", rpc: rpc, svcErr: fmt.Errorf("service call: %w", s.internal), want: s.want, wantCode: s.wantCode},
			)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			tt.rpc.setErr(fake, tt.svcErr)
			srv := newTestServer(fake)

			respIsNil, err := tt.rpc.call(t, srv)

			require.ErrorIs(t, err, tt.want)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.True(t, respIsNil)
		})
	}
}

// TestAccountServer_StatusPassthrough проверяет, что код gRPC-статуса от сервиса сохраняется.
func TestAccountServer_StatusPassthrough(t *testing.T) {
	statusErrors := []struct {
		name     string
		svcErr   error
		wantCode codes.Code
	}{
		{name: "unavailable", svcErr: status.Error(codes.Unavailable, "ledger down"), wantCode: codes.Unavailable},
		{name: "wrapped unavailable", svcErr: fmt.Errorf("close account: %w", status.Error(codes.Unavailable, "ledger down")), wantCode: codes.Unavailable},
		{name: "permission denied", svcErr: status.Error(codes.PermissionDenied, "denied"), wantCode: codes.PermissionDenied},
		{name: "not found", svcErr: status.Error(codes.NotFound, "upstream not found"), wantCode: codes.NotFound},
	}

	type passCase struct {
		name     string
		rpc      rpcCase
		svcErr   error
		wantCode codes.Code
	}
	var tests []passCase
	for _, rpc := range allRPCs() {
		for _, se := range statusErrors {
			tests = append(tests, passCase{name: rpc.name + "/" + se.name, rpc: rpc, svcErr: se.svcErr, wantCode: se.wantCode})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			tt.rpc.setErr(fake, tt.svcErr)
			srv := newTestServer(fake)

			respIsNil, err := tt.rpc.call(t, srv)

			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.True(t, respIsNil)
		})
	}
}

// TestAccountServer_UnknownErrorInvariant проверяет Internal-заглушку своего RPC на произвольной ошибке.
func TestAccountServer_UnknownErrorInvariant(t *testing.T) {
	unknownErrors := []struct {
		name   string
		svcErr error
	}{
		{name: "db failure", svcErr: errors.New("db failure")},
		{name: "wrapped secret detail", svcErr: fmt.Errorf("wrap: %w", errors.New("secret detail"))},
	}

	type unknownCase struct {
		name   string
		rpc    rpcCase
		svcErr error
	}
	var tests []unknownCase
	for _, rpc := range allRPCs() {
		for _, ue := range unknownErrors {
			tests = append(tests, unknownCase{name: rpc.name + "/" + ue.name, rpc: rpc, svcErr: ue.svcErr})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			tt.rpc.setErr(fake, tt.svcErr)
			srv := newTestServer(fake)

			respIsNil, err := tt.rpc.call(t, srv)

			require.ErrorIs(t, err, tt.rpc.stubError)
			assert.Equal(t, codes.Internal, status.Code(err))
			// текст исходной ошибки не утекает: сообщение равно тексту заглушки
			assert.Equal(t, status.Convert(tt.rpc.stubError).Message(), status.Convert(err).Message())
			assert.True(t, respIsNil)
		})
	}
}

// TestAccountServer_CallsOwnMethodOnce проверяет, что валидный запрос вызывает ровно свой метод сервиса один раз.
func TestAccountServer_CallsOwnMethodOnce(t *testing.T) {
	for _, tt := range allRPCs() {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{
				openAccount:    newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
				getAccount:     newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
				getByNumber:    newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
				blockAccount:   newAccountModel(TEST_ACCOUNT_ID, models.STATUS_BLOCKED),
				unblockAccount: newAccountModel(TEST_ACCOUNT_ID, models.STATUS_ACTIVE),
				closeAccount:   newAccountModel(TEST_ACCOUNT_ID, models.STATUS_CLOSED),
				listAccounts:   []*models.Account{},
				balance:        &models.Balance{AccountID: uuid.MustParse(TEST_ACCOUNT_ID), Amount: 0, Currency: TEST_CURRENCY},
				history:        []*models.StatusChange{},
			}
			srv := newTestServer(fake)

			respIsNil, err := tt.call(t, srv)

			require.NoError(t, err)
			assert.False(t, respIsNil)
			assert.Equal(t, 1, tt.ownCalls(fake))
			assert.Equal(t, 1, fake.totalCalls())
		})
	}
}
