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
	"google.golang.org/protobuf/proto"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

const (
	TEST_INITIATOR_ID    = "0b9d6c7e-1f2a-4c3b-8d4e-5f6a7b8c9d01"
	TEST_FROM_ACCOUNT_ID = "1c8e7d6f-2a3b-4d4c-9e5f-6a7b8c9d0e02"
	TEST_TO_ACCOUNT_ID   = "2d7f8e9a-3b4c-4e5d-af60-7b8c9d0e1f03"
	TEST_OFFICER_ID      = "3e6a9f0b-4c5d-4f6e-b071-8c9d0e1f2a04"
	TEST_AMOUNT          = int64(100)
	TEST_CURRENCY        = "RUB"
	TEST_RULE_CODE       = "velocity"
	TEST_FULL_METHOD     = "/antifraud.v1.AntifraudService/CheckTransfer"
	TEST_CHECK_ID        = "4f5b0a1c-5d6e-4a7f-8182-9d0e1f2a3b05"
)

// testCtxKey - ключ значения, по которому видно, что хендлер получил тот же ctx.
type testCtxKey struct{}

// validTransferReq отдаёт новый валидный CheckTransferRequest.
func validTransferReq() *antifraudv1.CheckTransferRequest {
	return &antifraudv1.CheckTransferRequest{
		InitiatorId:   TEST_INITIATOR_ID,
		FromAccountId: TEST_FROM_ACCOUNT_ID,
		ToAccountId:   TEST_TO_ACCOUNT_ID,
		Amount:        TEST_AMOUNT,
		Currency:      TEST_CURRENCY,
	}
}

// transferWith отдаёт валидный CheckTransferRequest после правки mutate.
func transferWith(mutate func(r *antifraudv1.CheckTransferRequest)) *antifraudv1.CheckTransferRequest {
	req := validTransferReq()
	mutate(req)
	return req
}

// validUpdateRuleReq отдаёт новый валидный UpdateRuleRequest с enabled=true.
func validUpdateRuleReq() *antifraudv1.UpdateRuleRequest {
	return &antifraudv1.UpdateRuleRequest{
		Code:      TEST_RULE_CODE,
		Enabled:   proto.Bool(true),
		OfficerId: TEST_OFFICER_ID,
	}
}

// validUpdateRuleParamsReq отдаёт новый валидный UpdateRuleRequest только с params.
func validUpdateRuleParamsReq() *antifraudv1.UpdateRuleRequest {
	return &antifraudv1.UpdateRuleRequest{
		Code:      TEST_RULE_CODE,
		Params:    &antifraudv1.RuleParams{MaxCount: 5, WindowSeconds: 60},
		OfficerId: TEST_OFFICER_ID,
	}
}

// updateRuleWith отдаёт валидный UpdateRuleRequest (enabled=true) после правки mutate.
func updateRuleWith(mutate func(r *antifraudv1.UpdateRuleRequest)) *antifraudv1.UpdateRuleRequest {
	req := validUpdateRuleReq()
	mutate(req)
	return req
}

func TestValidate_CheckTransfer(t *testing.T) {
	tests := []struct {
		name    string
		req     *antifraudv1.CheckTransferRequest
		wantErr error
	}{
		{name: "valid request", req: validTransferReq()},
		{name: "amount one", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = 1 })},
		{name: "amount max int64", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = math.MaxInt64 })},
		{name: "amount zero", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = 0 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "amount minus one", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = -1 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "amount min int64", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = math.MinInt64 }), wantErr: domain.ErrAmountMustBePositive},
		{name: "initiator empty", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.InitiatorId = "" }), wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "initiator spaces", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.InitiatorId = "   " }), wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "initiator mixed whitespace", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.InitiatorId = "\t\n " }), wantErr: domain.ErrInitiatorIDMustBeUUID},
		{name: "from account empty", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.FromAccountId = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "from account tab", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.FromAccountId = "\t" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "to account empty", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.ToAccountId = "" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "to account newline", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.ToAccountId = "\n" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "to account crlf", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.ToAccountId = "\r\n" }), wantErr: domain.ErrAccountIDMustBeUUID},
		{name: "currency empty", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Currency = "" }), wantErr: domain.ErrCurrencyIsRequired},
		{name: "currency whitespace", req: transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Currency = "  \t" }), wantErr: domain.ErrCurrencyIsRequired},
		{
			name: "bad formats pass through",
			req: &antifraudv1.CheckTransferRequest{
				InitiatorId:   "not-a-uuid",
				FromAccountId: "not-a-uuid",
				ToAccountId:   "not-a-uuid",
				Amount:        TEST_AMOUNT,
				Currency:      "xx",
			},
		},
		{
			name: "values with surrounding spaces pass through",
			req: &antifraudv1.CheckTransferRequest{
				InitiatorId:   " " + TEST_INITIATOR_ID + " ",
				FromAccountId: " " + TEST_FROM_ACCOUNT_ID,
				ToAccountId:   TEST_TO_ACCOUNT_ID + "\t",
				Amount:        TEST_AMOUNT,
				Currency:      " RUB ",
			},
		},
		{name: "zero message reports initiator first", req: &antifraudv1.CheckTransferRequest{}, wantErr: domain.ErrInitiatorIDMustBeUUID},
		{
			name: "initiator before amount",
			req: transferWith(func(r *antifraudv1.CheckTransferRequest) {
				r.InitiatorId = ""
				r.Amount = 0
			}),
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name: "from and to empty",
			req: transferWith(func(r *antifraudv1.CheckTransferRequest) {
				r.FromAccountId = ""
				r.ToAccountId = ""
			}),
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "to account before currency",
			req: transferWith(func(r *antifraudv1.CheckTransferRequest) {
				r.ToAccountId = ""
				r.Currency = ""
			}),
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "to account before amount",
			req: transferWith(func(r *antifraudv1.CheckTransferRequest) {
				r.ToAccountId = ""
				r.Amount = -1
			}),
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name: "amount before currency",
			req: transferWith(func(r *antifraudv1.CheckTransferRequest) {
				r.Amount = 0
				r.Currency = ""
			}),
			wantErr: domain.ErrAmountMustBePositive,
		},
	}

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

func TestValidate_UpdateRule(t *testing.T) {
	tests := []struct {
		name    string
		req     *antifraudv1.UpdateRuleRequest
		wantErr error
	}{
		{name: "enabled true", req: validUpdateRuleReq()},
		{name: "enabled false counts as set", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Enabled = proto.Bool(false) })},
		{
			name: "empty params counts as set",
			req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) {
				r.Enabled = nil
				r.Params = &antifraudv1.RuleParams{}
			}),
		},
		{
			name: "filled params",
			req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) {
				r.Enabled = nil
				r.Params = &antifraudv1.RuleParams{Limits: map[string]int64{"RUB": 1000}, MaxCount: 5, WindowSeconds: 60}
			}),
		},
		{name: "enabled and params", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Params = &antifraudv1.RuleParams{} })},
		{name: "code empty", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Code = "" }), wantErr: domain.ErrRuleCodeIsRequired},
		{name: "code whitespace", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Code = " \t\n" }), wantErr: domain.ErrRuleCodeIsRequired},
		{name: "code crlf", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Code = "\r\n" }), wantErr: domain.ErrRuleCodeIsRequired},
		{name: "nothing to update", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Enabled = nil }), wantErr: domain.ErrNothingToUpdate},
		{name: "officer empty", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.OfficerId = "" }), wantErr: domain.ErrOfficerIDMustBeUUID},
		{
			name:    "officer spaces with empty params",
			req:     &antifraudv1.UpdateRuleRequest{Code: TEST_RULE_CODE, Params: &antifraudv1.RuleParams{}, OfficerId: "  "},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
		{name: "officer mixed whitespace", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.OfficerId = "\t\n" }), wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "officer not uuid passes through", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.OfficerId = "not-a-uuid" })},
		{name: "unknown rule code passes through", req: updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Code = "unknown_rule" })},
		{name: "code checked before nothing to update", req: &antifraudv1.UpdateRuleRequest{Code: "", OfficerId: ""}, wantErr: domain.ErrRuleCodeIsRequired},
		{name: "nothing to update before officer", req: &antifraudv1.UpdateRuleRequest{Code: TEST_RULE_CODE}, wantErr: domain.ErrNothingToUpdate},
		{name: "zero message reports code first", req: &antifraudv1.UpdateRuleRequest{}, wantErr: domain.ErrRuleCodeIsRequired},
	}

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

func TestValidate_OtherRequests(t *testing.T) {
	tests := []struct {
		name string
		req  any
	}{
		{name: "list rules request", req: &antifraudv1.ListRulesRequest{}},
		{name: "untyped nil", req: nil},
		{name: "arbitrary struct", req: struct{ Name string }{}},
		{name: "pointer to arbitrary struct", req: &struct{ Name string }{}},
		{name: "string", req: "x"},
		{name: "int", req: 42},
		{name: "rule params message", req: &antifraudv1.RuleParams{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, validate(tt.req))
		})
	}
}

// Инвариант: обнуление ровно одного обязательного поля валидного запроса даёт ошибку этого поля.
func TestValidate_RequiredFieldInvariant(t *testing.T) {
	tests := []struct {
		name      string
		newValid  func() any
		newBroken func() any
		wantErr   error
	}{
		{
			name:      "transfer initiator",
			newValid:  func() any { return validTransferReq() },
			newBroken: func() any { return transferWith(func(r *antifraudv1.CheckTransferRequest) { r.InitiatorId = "" }) },
			wantErr:   domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:      "transfer from account",
			newValid:  func() any { return validTransferReq() },
			newBroken: func() any { return transferWith(func(r *antifraudv1.CheckTransferRequest) { r.FromAccountId = "" }) },
			wantErr:   domain.ErrAccountIDMustBeUUID,
		},
		{
			name:      "transfer to account",
			newValid:  func() any { return validTransferReq() },
			newBroken: func() any { return transferWith(func(r *antifraudv1.CheckTransferRequest) { r.ToAccountId = "" }) },
			wantErr:   domain.ErrAccountIDMustBeUUID,
		},
		{
			name:      "transfer amount",
			newValid:  func() any { return validTransferReq() },
			newBroken: func() any { return transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = 0 }) },
			wantErr:   domain.ErrAmountMustBePositive,
		},
		{
			name:      "transfer currency",
			newValid:  func() any { return validTransferReq() },
			newBroken: func() any { return transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Currency = "" }) },
			wantErr:   domain.ErrCurrencyIsRequired,
		},
		{
			name:      "update rule code",
			newValid:  func() any { return validUpdateRuleReq() },
			newBroken: func() any { return updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Code = "" }) },
			wantErr:   domain.ErrRuleCodeIsRequired,
		},
		{
			name:      "update rule enabled",
			newValid:  func() any { return validUpdateRuleReq() },
			newBroken: func() any { return updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Enabled = nil }) },
			wantErr:   domain.ErrNothingToUpdate,
		},
		{
			name:     "update rule params",
			newValid: func() any { return validUpdateRuleParamsReq() },
			newBroken: func() any {
				req := validUpdateRuleParamsReq()
				req.Params = nil
				return req
			},
			wantErr: domain.ErrNothingToUpdate,
		},
		{
			name:      "update rule officer",
			newValid:  func() any { return validUpdateRuleReq() },
			newBroken: func() any { return updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.OfficerId = "" }) },
			wantErr:   domain.ErrOfficerIDMustBeUUID,
		},
		{
			name:     "update rule params base officer",
			newValid: func() any { return validUpdateRuleParamsReq() },
			newBroken: func() any {
				req := validUpdateRuleParamsReq()
				req.OfficerId = ""
				return req
			},
			wantErr: domain.ErrOfficerIDMustBeUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, validate(tt.newValid()))

			err := validate(tt.newBroken())

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestValidateInterceptor_RejectsInvalid(t *testing.T) {
	tests := []struct {
		name    string
		req     any
		wantErr error
	}{
		{
			name:    "transfer amount zero",
			req:     transferWith(func(r *antifraudv1.CheckTransferRequest) { r.Amount = 0 }),
			wantErr: domain.ErrAmountMustBePositive,
		},
		{
			name:    "update rule nothing to update",
			req:     updateRuleWith(func(r *antifraudv1.UpdateRuleRequest) { r.Enabled = nil }),
			wantErr: domain.ErrNothingToUpdate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &fakeHandler{resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID}}
			info := &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}

			resp, err := Validate()(t.Context(), tt.req, info, handler.handle)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, resp)
			assert.Equal(t, []handlerCall(nil), handler.calls)
		})
	}
}

func TestValidateInterceptor_PassesThrough(t *testing.T) {
	// Ошибки хендлера, проброс которых проверяется по errors.Is.
	handlerErr := status.Error(codes.Internal, "handler failed")
	wrappedSentinel := errors.New("handler sentinel")

	tests := []struct {
		name        string
		newCtx      func(t *testing.T) context.Context
		req         any
		info        *grpc.UnaryServerInfo
		handlerResp any
		handlerErr  error
	}{
		{
			name:        "valid transfer response forwarded",
			newCtx:      func(t *testing.T) context.Context { return context.WithValue(t.Context(), testCtxKey{}, "transfer") },
			req:         validTransferReq(),
			info:        &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD},
			handlerResp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW},
		},
		{
			name:       "handler error forwarded",
			newCtx:     func(t *testing.T) context.Context { return context.WithValue(t.Context(), testCtxKey{}, "error") },
			req:        validUpdateRuleReq(),
			info:       &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD},
			handlerErr: handlerErr,
		},
		{
			name:        "handler response and error both forwarded",
			newCtx:      func(t *testing.T) context.Context { return context.WithValue(t.Context(), testCtxKey{}, "both") },
			req:         validTransferReq(),
			info:        &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD},
			handlerResp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID},
			handlerErr:  wrappedSentinel,
		},
		{
			name:        "list rules passes",
			newCtx:      func(t *testing.T) context.Context { return context.WithValue(t.Context(), testCtxKey{}, "list") },
			req:         &antifraudv1.ListRulesRequest{},
			info:        &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD},
			handlerResp: &antifraudv1.ListRulesResponse{},
		},
		{
			name:        "nil request passes",
			newCtx:      func(t *testing.T) context.Context { return context.WithValue(t.Context(), testCtxKey{}, "nil") },
			req:         nil,
			info:        &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD},
			handlerResp: &antifraudv1.ListRulesResponse{},
		},
		{
			name:        "nil info",
			newCtx:      func(t *testing.T) context.Context { return context.WithValue(t.Context(), testCtxKey{}, "nil info") },
			req:         validTransferReq(),
			info:        nil,
			handlerResp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID},
		},
		{
			name: "canceled context forwarded as is",
			newCtx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			req:         validTransferReq(),
			info:        &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD},
			handlerResp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &fakeHandler{resp: tt.handlerResp, err: tt.handlerErr}
			ctx := tt.newCtx(t)

			resp, err := Validate()(ctx, tt.req, tt.info, handler.handle)

			assert.ErrorIs(t, err, tt.handlerErr)
			assert.Equal(t, tt.handlerResp, resp)
			assert.Equal(t, []handlerCall{{ctx: ctx, req: tt.req}}, handler.calls)
		})
	}
}
