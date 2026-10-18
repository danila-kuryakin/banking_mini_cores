package client

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/antifraud/v1"
)

const (
	TEST_INITIATOR_ID  = "7c4e1f2a-3b5d-4c6e-9f80-a1b2c3d4e501"
	TEST_FROM_ACCOUNT  = "7c4e1f2a-3b5d-4c6e-9f80-a1b2c3d4e502"
	TEST_TO_ACCOUNT    = "7c4e1f2a-3b5d-4c6e-9f80-a1b2c3d4e503"
	TEST_CHECK_ID      = "af-check-1"
	TEST_AMOUNT        = int64(150_00)
	TEST_REASON_VELO   = "velocity"
	TEST_REASON_DAILY  = "daily_amount_limit"
	TEST_UNKNOWN_VALUE = 99
)

// testTransferCommand - перевод с фиксированными идентификаторами
func testTransferCommand() models.TransferCommand {
	return models.TransferCommand{
		InitiatorID: uuid.MustParse(TEST_INITIATOR_ID),
		From:        uuid.MustParse(TEST_FROM_ACCOUNT),
		To:          uuid.MustParse(TEST_TO_ACCOUNT),
		Amount:      TEST_AMOUNT,
		Currency:    TEST_CURRENCY_RUB,
	}
}

func TestAntifraudClient_CheckTransfer(t *testing.T) {
	tests := []struct {
		name string
		resp *antifraudv1.CheckTransferResponse
		want models.Verdict
	}{
		{
			name: "allow drops reasons",
			resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW, Reasons: []string{TEST_REASON_VELO}},
			want: models.Verdict{Allowed: true, CheckID: TEST_CHECK_ID, Reasons: nil},
		},
		{
			name: "deny keeps reasons",
			resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_DENY, Reasons: []string{TEST_REASON_VELO, TEST_REASON_DAILY}},
			want: models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID, Reasons: []string{TEST_REASON_VELO, TEST_REASON_DAILY}},
		},
		{
			name: "deny without reasons",
			resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_DENY},
			want: models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID, Reasons: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAntifraudServiceClient{resp: tt.resp}
			c := NewAntifraudClient(fake)

			got, err := c.CheckTransfer(t.Context(), testTransferCommand())

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAntifraudClient_CheckTransfer_Request(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
	}{
		{name: "regular", amount: TEST_AMOUNT, currency: TEST_CURRENCY_RUB},
		{name: "currency passed as is", amount: TEST_AMOUNT, currency: " rub "},
		{name: "zero amount passed as is", amount: 0, currency: TEST_CURRENCY_RUB},
		{name: "negative amount passed as is", amount: -5, currency: TEST_CURRENCY_RUB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAntifraudServiceClient{resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW}}
			c := NewAntifraudClient(fake)
			cmd := testTransferCommand()
			cmd.Amount = tt.amount
			cmd.Currency = tt.currency

			_, err := c.CheckTransfer(t.Context(), cmd)

			require.NoError(t, err)
			want := &antifraudv1.CheckTransferRequest{
				InitiatorId:   TEST_INITIATOR_ID,
				FromAccountId: TEST_FROM_ACCOUNT,
				ToAccountId:   TEST_TO_ACCOUNT,
				Amount:        tt.amount,
				Currency:      tt.currency,
			}
			require.Len(t, fake.requests, 1)
			assert.True(t, proto.Equal(want, fake.requests[0]), "unexpected request: %v", fake.requests[0])
		})
	}
}

func TestAntifraudClient_CheckTransfer_UnknownVerdict(t *testing.T) {
	tests := []struct {
		name    string
		verdict antifraudv1.Verdict
	}{
		{name: "unspecified", verdict: antifraudv1.Verdict_VERDICT_UNSPECIFIED},
		{name: "unknown value", verdict: antifraudv1.Verdict(TEST_UNKNOWN_VALUE)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAntifraudServiceClient{resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: tt.verdict, Reasons: []string{TEST_REASON_VELO}}}
			c := NewAntifraudClient(fake)

			got, err := c.CheckTransfer(t.Context(), testTransferCommand())

			require.Error(t, err)
			assert.Equal(t, models.Verdict{}, got)
		})
	}
}

func TestAntifraudClient_CheckTransfer_RPCError(t *testing.T) {
	tests := []struct {
		name   string
		rpcErr error
		newCtx func(t *testing.T) context.Context
	}{
		{name: "unavailable", rpcErr: status.Error(codes.Unavailable, "connection refused"), newCtx: liveContext},
		{name: "deadline exceeded", rpcErr: status.Error(codes.DeadlineExceeded, "deadline"), newCtx: liveContext},
		{name: "canceled context", rpcErr: status.Error(codes.Canceled, "context canceled"), newCtx: canceledContext},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAntifraudServiceClient{err: tt.rpcErr}
			c := NewAntifraudClient(fake)

			got, err := c.CheckTransfer(tt.newCtx(t), testTransferCommand())

			require.ErrorIs(t, err, tt.rpcErr)
			assert.Equal(t, models.Verdict{}, got)
		})
	}
}

// инвариант: Allowed выставляется только при явном ALLOW
func TestAntifraudClient_CheckTransfer_AllowedOnlyOnAllow(t *testing.T) {
	tests := []struct {
		name        string
		resp        *antifraudv1.CheckTransferResponse
		rpcErr      error
		wantAllowed bool
	}{
		{name: "allow", resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW}, wantAllowed: true},
		{name: "deny", resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_DENY}},
		{name: "unspecified", resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_UNSPECIFIED}},
		{name: "unknown value", resp: &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict(TEST_UNKNOWN_VALUE)}},
		{name: "rpc error", rpcErr: status.Error(codes.Unavailable, "connection refused")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAntifraudServiceClient{resp: tt.resp, err: tt.rpcErr}
			c := NewAntifraudClient(fake)

			got, err := c.CheckTransfer(t.Context(), testTransferCommand())

			assert.Equal(t, tt.wantAllowed, got.Allowed, "err: %v", err)
		})
	}
}
