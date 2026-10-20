package interceptors

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

const (
	TEST_INITIATOR_ID   = "0b7d3c1a-6f2e-4a8b-9c1d-2e3f4a5b6c01"
	TEST_FROM_ACCOUNT   = "0b7d3c1a-6f2e-4a8b-9c1d-2e3f4a5b6c02"
	TEST_TO_ACCOUNT     = "0b7d3c1a-6f2e-4a8b-9c1d-2e3f4a5b6c03"
	TEST_TO_NUMBER      = "40817810400000000014"
	TEST_OFFICER_ID     = "0b7d3c1a-6f2e-4a8b-9c1d-2e3f4a5b6c04"
	TEST_TRANSACTION_ID = "0b7d3c1a-6f2e-4a8b-9c1d-2e3f4a5b6c05"
	TEST_USER_ID        = "0b7d3c1a-6f2e-4a8b-9c1d-2e3f4a5b6c06"
	TEST_CURRENCY       = "RUB"
	TEST_AMOUNT         = int64(100)
	TEST_NOT_UUID       = "abc"
	TEST_HANDLER_RESP   = "handler-response"
	TEST_FULL_METHOD    = "/ledger.v1.LedgerService/Method"
)

// ctxMarkerKey - ключ значения, по которому видно, что хендлер получил тот же ctx
type ctxMarkerKey struct{}

// recordingHandler записывает вызовы хендлера и отдаёт заданный ответ
type recordingHandler struct {
	calls  int
	gotCtx context.Context
	gotReq any
	resp   any
	err    error
}

func (h *recordingHandler) handle(ctx context.Context, req any) (any, error) {
	h.calls++
	h.gotCtx = ctx
	h.gotReq = req
	return h.resp, h.err
}

func transfer(mutate func(r *ledgerv1.TransferRequest)) *ledgerv1.TransferRequest {
	r := &ledgerv1.TransferRequest{
		InitiatorId:   TEST_INITIATOR_ID,
		FromAccountId: TEST_FROM_ACCOUNT,
		ToAccountId:   TEST_TO_ACCOUNT,
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY,
	}
	mutate(r)
	return r
}

func deposit(mutate func(r *ledgerv1.DepositRequest)) *ledgerv1.DepositRequest {
	r := &ledgerv1.DepositRequest{
		AccountId: TEST_TO_ACCOUNT,
		Amount:    TEST_AMOUNT,
		Currency:  TEST_CURRENCY,
		OfficerId: TEST_OFFICER_ID,
	}
	mutate(r)
	return r
}

func listTransactions(mutate func(r *ledgerv1.ListTransactionsRequest)) *ledgerv1.ListTransactionsRequest {
	r := &ledgerv1.ListTransactionsRequest{
		UserId:    TEST_USER_ID,
		AccountId: TEST_FROM_ACCOUNT,
		Limit:     20,
		Offset:    0,
	}
	mutate(r)
	return r
}

// testStruct - произвольный тип запроса, о котором интерцептор не знает
type testStruct struct {
	Name string
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     any
		wantErr error
	}{
		// TransferRequest
		{name: "transfer valid", req: transfer(func(r *ledgerv1.TransferRequest) {})},
		{name: "transfer empty initiator", req: transfer(func(r *ledgerv1.TransferRequest) { r.InitiatorId = "" }), wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "transfer blank initiator", req: transfer(func(r *ledgerv1.TransferRequest) { r.InitiatorId = " \t" }), wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "transfer empty from", req: transfer(func(r *ledgerv1.TransferRequest) { r.FromAccountId = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "transfer blank from", req: transfer(func(r *ledgerv1.TransferRequest) { r.FromAccountId = "   " }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "transfer empty to", req: transfer(func(r *ledgerv1.TransferRequest) { r.ToAccountId = "" }), wantErr: domain.ErrRecipientIsRequired},
		{name: "transfer blank to and number", req: transfer(func(r *ledgerv1.TransferRequest) { r.ToAccountId = " "; r.ToAccountNumber = "\t" }), wantErr: domain.ErrRecipientIsRequired},
		{name: "transfer to and number both set", req: transfer(func(r *ledgerv1.TransferRequest) { r.ToAccountNumber = TEST_TO_NUMBER }), wantErr: domain.ErrRecipientIsAmbiguous},
		{name: "transfer number only", req: transfer(func(r *ledgerv1.TransferRequest) { r.ToAccountId = ""; r.ToAccountNumber = TEST_TO_NUMBER })},
		{name: "transfer malformed number passes", req: transfer(func(r *ledgerv1.TransferRequest) { r.ToAccountId = ""; r.ToAccountNumber = TEST_NOT_UUID })},
		{name: "transfer recipient before amount", req: transfer(func(r *ledgerv1.TransferRequest) { r.ToAccountId = ""; r.Amount = 0 }), wantErr: domain.ErrRecipientIsRequired},
		{name: "transfer zero amount", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = 0 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "transfer negative amount", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = -1 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "transfer min int64 amount", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = math.MinInt64 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "transfer amount one", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = 1 })},
		{name: "transfer max int64 amount", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = math.MaxInt64 })},
		{name: "transfer empty currency", req: transfer(func(r *ledgerv1.TransferRequest) { r.Currency = "" }), wantErr: domain.ErrCurrencyIsRequired},
		{name: "transfer blank currency", req: transfer(func(r *ledgerv1.TransferRequest) { r.Currency = "  " }), wantErr: domain.ErrCurrencyIsRequired},
		{name: "transfer all invalid reports initiator first", req: &ledgerv1.TransferRequest{}, wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "transfer from before amount and currency", req: transfer(func(r *ledgerv1.TransferRequest) { r.FromAccountId = ""; r.Amount = 0; r.Currency = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "transfer amount before currency", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = 0; r.Currency = "" }), wantErr: domain.ErrAmountMustBePositive},
		{name: "transfer non uuid ids pass", req: transfer(func(r *ledgerv1.TransferRequest) {
			r.InitiatorId, r.FromAccountId, r.ToAccountId = TEST_NOT_UUID, TEST_NOT_UUID, TEST_NOT_UUID
		})},
		// DepositRequest
		{name: "deposit valid", req: deposit(func(r *ledgerv1.DepositRequest) {})},
		{name: "deposit empty account", req: deposit(func(r *ledgerv1.DepositRequest) { r.AccountId = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "deposit blank account", req: deposit(func(r *ledgerv1.DepositRequest) { r.AccountId = " " }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "deposit empty officer", req: deposit(func(r *ledgerv1.DepositRequest) { r.OfficerId = "" }), wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "deposit blank officer", req: deposit(func(r *ledgerv1.DepositRequest) { r.OfficerId = "\t" }), wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "deposit zero amount", req: deposit(func(r *ledgerv1.DepositRequest) { r.Amount = 0 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "deposit negative amount", req: deposit(func(r *ledgerv1.DepositRequest) { r.Amount = -1 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "deposit empty currency", req: deposit(func(r *ledgerv1.DepositRequest) { r.Currency = "" }), wantErr: domain.ErrCurrencyIsRequired},
		{name: "deposit blank currency", req: deposit(func(r *ledgerv1.DepositRequest) { r.Currency = "  " }), wantErr: domain.ErrCurrencyIsRequired},
		{name: "deposit all invalid reports account first", req: &ledgerv1.DepositRequest{}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "deposit non uuid ids pass", req: deposit(func(r *ledgerv1.DepositRequest) { r.AccountId, r.OfficerId = TEST_NOT_UUID, TEST_NOT_UUID })},
		// GetTransactionRequest
		{name: "get transaction valid", req: &ledgerv1.GetTransactionRequest{TransactionId: TEST_TRANSACTION_ID}},
		{name: "get transaction empty id", req: &ledgerv1.GetTransactionRequest{TransactionId: ""}, wantErr: domain.ErrTransactionIDMustBeUUID},
		{name: "get transaction blank id", req: &ledgerv1.GetTransactionRequest{TransactionId: " "}, wantErr: domain.ErrTransactionIDMustBeUUID},
		{name: "get transaction non uuid passes", req: &ledgerv1.GetTransactionRequest{TransactionId: TEST_NOT_UUID}},
		// ListTransactionsRequest
		{name: "list zero page without user", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.UserId = ""; r.Limit = 0; r.Offset = 0 })},
		{name: "list with user and page", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.Limit = 100; r.Offset = 5 })},
		{name: "list empty account", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.AccountId = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "list blank account", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.AccountId = " " }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "list negative limit", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.Limit = -1 }), wantErr: domain.ErrLimitIsNegative},
		{name: "list negative offset", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.Offset = -1 }), wantErr: domain.ErrOffsetIsNegative},
		{name: "list account before page", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.AccountId = ""; r.Limit = -1; r.Offset = -1 }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "list limit before offset", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.Limit = -1; r.Offset = -1 }), wantErr: domain.ErrLimitIsNegative},
		// GetBalanceRequest
		{name: "get balance valid", req: &ledgerv1.GetBalanceRequest{AccountId: TEST_FROM_ACCOUNT}},
		{name: "get balance empty account", req: &ledgerv1.GetBalanceRequest{AccountId: ""}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "get balance blank account", req: &ledgerv1.GetBalanceRequest{AccountId: " "}, wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "get balance non uuid passes", req: &ledgerv1.GetBalanceRequest{AccountId: TEST_NOT_UUID}},
		// прочие типы запросов
		{name: "nil request passes", req: nil},
		{name: "empty proto passes", req: &emptypb.Empty{}},
		{name: "unknown struct passes", req: &testStruct{Name: "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &recordingHandler{resp: TEST_HANDLER_RESP}
			ctx := context.WithValue(t.Context(), ctxMarkerKey{}, tt.name)
			info := &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}

			resp, err := Validate()(ctx, tt.req, info, handler.handle)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				assert.Nil(t, resp)
				assert.Equal(t, 0, handler.calls)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TEST_HANDLER_RESP, resp)
			assert.Equal(t, 1, handler.calls)
			assert.Equal(t, ctx, handler.gotCtx)
			assert.Equal(t, tt.req, handler.gotReq)
		})
	}
}

func TestValidate_ReturnsHandlerErrorAndResponse(t *testing.T) {
	errHandler := errors.New("handler failed")
	handler := &recordingHandler{resp: TEST_HANDLER_RESP, err: errHandler}
	req := transfer(func(r *ledgerv1.TransferRequest) {})

	resp, err := Validate()(t.Context(), req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, handler.handle)

	require.ErrorIs(t, err, errHandler)
	assert.Equal(t, TEST_HANDLER_RESP, resp)
	assert.Equal(t, 1, handler.calls)
	assert.Equal(t, req, handler.gotReq)
}

// инвариант: при любой ошибке проверки хендлер не вызывается
func TestValidate_InvalidRequestNeverReachesHandler(t *testing.T) {
	tests := []struct {
		name    string
		req     any
		wantErr error
	}{
		{name: "transfer empty initiator", req: transfer(func(r *ledgerv1.TransferRequest) { r.InitiatorId = "" }), wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "transfer empty from", req: transfer(func(r *ledgerv1.TransferRequest) { r.FromAccountId = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "transfer negative amount", req: transfer(func(r *ledgerv1.TransferRequest) { r.Amount = -1 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "transfer blank currency", req: transfer(func(r *ledgerv1.TransferRequest) { r.Currency = "  " }), wantErr: domain.ErrCurrencyIsRequired},
		{name: "deposit empty officer", req: deposit(func(r *ledgerv1.DepositRequest) { r.OfficerId = "" }), wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "get transaction empty id", req: &ledgerv1.GetTransactionRequest{}, wantErr: domain.ErrTransactionIDMustBeUUID},
		{name: "list negative limit", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.Limit = -1 }), wantErr: domain.ErrLimitIsNegative},
		{name: "list negative offset", req: listTransactions(func(r *ledgerv1.ListTransactionsRequest) { r.Offset = -1 }), wantErr: domain.ErrOffsetIsNegative},
		{name: "get balance empty account", req: &ledgerv1.GetBalanceRequest{}, wantErr: domain.ErrAccountIDMustBeUUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &recordingHandler{resp: TEST_HANDLER_RESP}

			resp, err := Validate()(t.Context(), tt.req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, handler.handle)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, resp)
			assert.Equal(t, 0, handler.calls)
		})
	}
}
