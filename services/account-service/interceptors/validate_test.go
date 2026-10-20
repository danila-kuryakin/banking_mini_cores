package interceptors

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
)

const (
	TEST_USER_ID      = "11111111-1111-1111-1111-111111111111"
	TEST_ACCOUNT_ID   = "22222222-2222-2222-2222-222222222222"
	TEST_OFFICER_ID   = "33333333-3333-3333-3333-333333333333"
	TEST_NOT_UUID     = "abc"
	TEST_CURRENCY     = "RUB"
	TEST_REASON       = "fraud suspicion"
	TEST_COMMENT      = "checked"
	TEST_LIMIT        = 20
	TEST_FULL_METHOD  = "/account.v1.AccountService/Test"
	TEST_REQUEST_MARK = "request-mark"
)

// Ключ контекста, по которому тест узнаёт свой ctx в обработчике.
type ctxMarkKey struct{}

type validateCase struct {
	name    string
	req     any
	wantErr error
}

// runValidateCases прогоняет таблицу кейсов через validate.
func runValidateCases(t *testing.T, tests []validateCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

// Строка из n символов "a" (n рун, n байт).
func latin(n int) string { return strings.Repeat("a", n) }

// Строка из n символов "я" (n рун, 2n байт).
func cyrillic(n int) string { return strings.Repeat("я", n) }

func TestValidate_OpenAccount(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: TEST_CURRENCY}},
		{name: "empty user_id", req: &accountv1.OpenAccountRequest{UserId: "", Currency: TEST_CURRENCY}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "whitespace user_id", req: &accountv1.OpenAccountRequest{UserId: " \t ", Currency: TEST_CURRENCY}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty currency passes", req: &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: ""}},
		{name: "unknown currency passes", req: &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: "ZZZZ"}},
		{name: "non-uuid user_id passes", req: &accountv1.OpenAccountRequest{UserId: TEST_NOT_UUID, Currency: TEST_CURRENCY}},
	})
}

func TestValidate_GetAccount(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}},
		{name: "empty user_id passes", req: &accountv1.GetAccountRequest{UserId: "", AccountId: TEST_ACCOUNT_ID}},
		{name: "empty account_id", req: &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "whitespace account_id", req: &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: "  "}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "both empty", req: &accountv1.GetAccountRequest{UserId: "", AccountId: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "non-uuid ids pass", req: &accountv1.GetAccountRequest{UserId: TEST_NOT_UUID, AccountId: TEST_NOT_UUID}},
	})
}

func TestValidate_ListAccounts(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: TEST_LIMIT, Offset: 0}},
		{name: "empty user_id", req: &accountv1.ListAccountsRequest{UserId: "", Limit: 0, Offset: 0}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "tab user_id", req: &accountv1.ListAccountsRequest{UserId: "\t", Limit: 0, Offset: 0}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "negative limit", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: -1, Offset: 0}, wantErr: domain.ErrLimitIsNegative},
		{name: "negative offset", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 0, Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "limit checked before offset", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "user_id checked before page", req: &accountv1.ListAccountsRequest{UserId: "", Limit: -1, Offset: -1}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "zero page passes", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 0, Offset: 0}},
		{name: "limit above ceiling passes", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 101, Offset: 0}},
		{name: "max int32 page passes", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: math.MaxInt32, Offset: math.MaxInt32}},
		{name: "min int32 limit", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: math.MinInt32, Offset: 0}, wantErr: domain.ErrLimitIsNegative},
		{name: "min int32 offset", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: 0, Offset: math.MinInt32}, wantErr: domain.ErrOffsetIsNegative},
	})
}

func TestValidate_ListStatusHistory(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: TEST_LIMIT, Offset: 0}},
		{name: "empty account_id", req: &accountv1.ListStatusHistoryRequest{AccountId: "", Limit: 0, Offset: 0}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "whitespace account_id", req: &accountv1.ListStatusHistoryRequest{AccountId: "  ", Limit: 0, Offset: 0}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "negative limit", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: -1, Offset: 0}, wantErr: domain.ErrLimitIsNegative},
		{name: "negative offset", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 0, Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "limit checked before offset", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "account_id checked before page", req: &accountv1.ListStatusHistoryRequest{AccountId: "", Limit: -1, Offset: -1}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "zero page passes", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 0, Offset: 0}},
		{name: "max int32 page passes", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: math.MaxInt32, Offset: math.MaxInt32}},
	})
}

func TestValidate_BlockAccount(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: TEST_REASON}},
		{name: "all empty gives account_id first", req: &accountv1.BlockAccountRequest{AccountId: "", OfficerId: "", Reason: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "whitespace account_id", req: &accountv1.BlockAccountRequest{AccountId: "  ", OfficerId: TEST_OFFICER_ID, Reason: TEST_REASON}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "empty officer_id before reason", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: "", Reason: ""}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "tab officer_id", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: "\t", Reason: TEST_REASON}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "empty reason", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: ""}, wantErr: domain.ErrReasonIsRequired},
		{name: "spaces reason", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "   "}, wantErr: domain.ErrReasonIsRequired},
		{name: "mixed whitespace reason", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: " \t\n "}, wantErr: domain.ErrReasonIsRequired},
		{name: "reason 500 runes passes", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: latin(domain.REASON_MAX_LENGTH)}},
		{name: "reason 501 runes", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: latin(domain.REASON_MAX_LENGTH + 1)}, wantErr: domain.ErrReasonIsTooLong},
		// 500 рун «я» = 1000 байт, считать надо руны
		{name: "reason 500 cyrillic runes passes", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: cyrillic(domain.REASON_MAX_LENGTH)}},
		{name: "reason 501 cyrillic runes", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: cyrillic(domain.REASON_MAX_LENGTH + 1)}, wantErr: domain.ErrReasonIsTooLong},
		// пробел + 500 "a" = 501 руна, длина считается без обрезки
		{name: "leading space counted", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: " " + latin(domain.REASON_MAX_LENGTH)}, wantErr: domain.ErrReasonIsTooLong},
		{name: "officer_id checked before long reason", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: "", Reason: latin(domain.REASON_MAX_LENGTH + 1)}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "non-uuid ids pass", req: &accountv1.BlockAccountRequest{AccountId: TEST_NOT_UUID, OfficerId: TEST_NOT_UUID, Reason: TEST_REASON}},
	})
}

func TestValidate_UnblockAccount(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT}},
		{name: "empty comment passes", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: ""}},
		{name: "empty account_id", req: &accountv1.UnblockAccountRequest{AccountId: "", OfficerId: TEST_OFFICER_ID, Comment: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "whitespace account_id", req: &accountv1.UnblockAccountRequest{AccountId: " ", OfficerId: TEST_OFFICER_ID, Comment: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "empty officer_id", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: "", Comment: ""}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "account_id checked before officer_id and comment", req: &accountv1.UnblockAccountRequest{AccountId: "", OfficerId: "", Comment: latin(domain.REASON_MAX_LENGTH + 1)}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "spaces comment passes", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: "   "}},
		{name: "comment 500 runes passes", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: latin(domain.REASON_MAX_LENGTH)}},
		{name: "comment 501 runes", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: latin(domain.REASON_MAX_LENGTH + 1)}, wantErr: domain.ErrReasonIsTooLong},
		// 500 рун «я» = 1000 байт, считать надо руны
		{name: "comment 500 cyrillic runes passes", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: cyrillic(domain.REASON_MAX_LENGTH)}},
		{name: "comment 501 cyrillic runes", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: cyrillic(domain.REASON_MAX_LENGTH + 1)}, wantErr: domain.ErrReasonIsTooLong},
		// пробел + 500 "a" = 501 руна, длина считается без обрезки
		{name: "leading space counted", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: " " + latin(domain.REASON_MAX_LENGTH)}, wantErr: domain.ErrReasonIsTooLong},
	})
}

func TestValidate_CloseAccount(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}},
		{name: "both empty gives user_id first", req: &accountv1.CloseAccountRequest{UserId: "", AccountId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty user_id", req: &accountv1.CloseAccountRequest{UserId: "", AccountId: TEST_ACCOUNT_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "tab user_id", req: &accountv1.CloseAccountRequest{UserId: "\t", AccountId: TEST_ACCOUNT_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty account_id", req: &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "whitespace account_id", req: &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: "  "}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "non-uuid ids pass", req: &accountv1.CloseAccountRequest{UserId: TEST_NOT_UUID, AccountId: TEST_NOT_UUID}},
	})
}

func TestValidate_GetBalance(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "valid", req: &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}},
		{name: "both empty gives user_id first", req: &accountv1.GetBalanceRequest{UserId: "", AccountId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty user_id", req: &accountv1.GetBalanceRequest{UserId: "", AccountId: TEST_ACCOUNT_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "tab user_id", req: &accountv1.GetBalanceRequest{UserId: "\t", AccountId: TEST_ACCOUNT_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty account_id", req: &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "whitespace account_id", req: &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: "  "}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "non-uuid ids pass", req: &accountv1.GetBalanceRequest{UserId: TEST_NOT_UUID, AccountId: TEST_NOT_UUID}},
	})
}

func TestValidate_UnknownRequest(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "empty proto", req: &emptypb.Empty{}},
		{name: "nil", req: nil},
		{name: "not proto string", req: "not a request"},
	})
}

// Инвариант: полностью заполненный корректный запрос любого из 8 типов проходит.
func TestValidate_ValidRequestsPass(t *testing.T) {
	typical := []validateCase{
		{name: "typical open", req: &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: TEST_CURRENCY}},
		{name: "typical get", req: &accountv1.GetAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}},
		{name: "typical list accounts", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: TEST_LIMIT, Offset: 0}},
		{name: "typical list history", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: TEST_LIMIT, Offset: 0}},
		{name: "typical block", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: TEST_REASON}},
		{name: "typical unblock", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT}},
		{name: "typical close", req: &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}},
		{name: "typical balance", req: &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}},
	}
	edge := []validateCase{
		{name: "edge open", req: &accountv1.OpenAccountRequest{UserId: TEST_NOT_UUID, Currency: ""}},
		{name: "edge get", req: &accountv1.GetAccountRequest{UserId: "", AccountId: TEST_NOT_UUID}},
		{name: "edge list accounts", req: &accountv1.ListAccountsRequest{UserId: TEST_USER_ID, Limit: math.MaxInt32, Offset: math.MaxInt32}},
		{name: "edge list history", req: &accountv1.ListStatusHistoryRequest{AccountId: TEST_ACCOUNT_ID, Limit: 0, Offset: 0}},
		{name: "edge block", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: cyrillic(domain.REASON_MAX_LENGTH)}},
		{name: "edge unblock", req: &accountv1.UnblockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Comment: cyrillic(domain.REASON_MAX_LENGTH)}},
		{name: "edge close", req: &accountv1.CloseAccountRequest{UserId: TEST_NOT_UUID, AccountId: TEST_NOT_UUID}},
		{name: "edge balance", req: &accountv1.GetBalanceRequest{UserId: TEST_NOT_UUID, AccountId: TEST_NOT_UUID}},
	}
	runValidateCases(t, append(typical, edge...))
}

// Заглушка обработчика: запоминает вызовы и отдаёт заданные ответ и ошибку.
type stubHandler struct {
	resp   any
	err    error
	calls  int
	gotCtx context.Context
	gotReq any
}

func (h *stubHandler) handle(ctx context.Context, req any) (any, error) {
	h.calls++
	h.gotCtx = ctx
	h.gotReq = req
	return h.resp, h.err
}

func TestValidateInterceptor_RejectsInvalid(t *testing.T) {
	tests := []struct {
		name    string
		req     any
		wantErr error
	}{
		{name: "close without user_id", req: &accountv1.CloseAccountRequest{UserId: "", AccountId: TEST_ACCOUNT_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "block with whitespace reason", req: &accountv1.BlockAccountRequest{AccountId: TEST_ACCOUNT_ID, OfficerId: TEST_OFFICER_ID, Reason: "  "}, wantErr: domain.ErrReasonIsRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &stubHandler{resp: &emptypb.Empty{}}
			info := &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}

			resp, err := Validate()(t.Context(), tt.req, info, handler.handle)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, resp)
			assert.Equal(t, 0, handler.calls)
		})
	}
}

func TestValidateInterceptor_PassesToHandler(t *testing.T) {
	handlerResp := &accountv1.GetBalanceRequest{UserId: TEST_USER_ID}
	handlerErr := errors.New("handler failed")
	tests := []struct {
		name        string
		req         any
		handlerResp any
		handlerErr  error
	}{
		{name: "response returned", req: &accountv1.CloseAccountRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}, handlerResp: handlerResp},
		{name: "handler error returned", req: &accountv1.GetBalanceRequest{UserId: TEST_USER_ID, AccountId: TEST_ACCOUNT_ID}, handlerErr: handlerErr},
		{name: "response and error returned", req: &accountv1.OpenAccountRequest{UserId: TEST_USER_ID, Currency: TEST_CURRENCY}, handlerResp: handlerResp, handlerErr: handlerErr},
		{name: "unknown request passed", req: &emptypb.Empty{}, handlerResp: handlerResp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &stubHandler{resp: tt.handlerResp, err: tt.handlerErr}
			info := &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}
			ctx := context.WithValue(t.Context(), ctxMarkKey{}, TEST_REQUEST_MARK)

			resp, err := Validate()(ctx, tt.req, info, handler.handle)

			assert.Equal(t, tt.handlerErr, err)
			assert.Equal(t, tt.handlerResp, resp)
			require.Equal(t, 1, handler.calls)
			assert.Equal(t, ctx, handler.gotCtx)
			assert.Equal(t, TEST_REQUEST_MARK, handler.gotCtx.Value(ctxMarkKey{}))
			assert.Same(t, tt.req, handler.gotReq)
		})
	}
}
