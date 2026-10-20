//go:build !gremlins

package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
)

// BUG-1: текст ErrCurrencyNotAllowed с вводом клиента не должен уходить наружу.
func TestLedgerServer_Bug_CurrencyNotAllowedHidesDetails(t *testing.T) {
	rpcs := allRPCs()[:2] // валюту принимают только Transfer и Deposit
	wantErr := status.Error(codes.InvalidArgument, TEST_CURRENCY_MESSAGE)
	tests := []struct {
		name string
		err  error
	}{
		{name: "wrapped with client input", err: fmt.Errorf("%w: currency \"rub1\": expected a three-letter ISO 4217 code", domain.ErrCurrencyNotAllowed)},
		{name: "wrapped with prefix", err: fmt.Errorf("transfer: %w", domain.ErrCurrencyNotAllowed)},
	}
	for _, rpc := range rpcs {
		for _, tt := range tests {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				fake := newFailingLedgerService(tt.err)
				srv, logBuf := newTestServer(fake)

				got, err := rpc.call(t.Context(), srv)

				require.ErrorIs(t, err, wantErr)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				assert.Equal(t, TEST_CURRENCY_MESSAGE, status.Convert(err).Message())
				assert.Nil(t, got)
				assert.False(t, strings.Contains(logBuf.String(), TEST_ERROR_LOG))
			})
		}
	}
}

// BUG-2: статус вышестоящего сервиса не пробрасывается, а становится Internal своего RPC.
func TestLedgerServer_Bug_UpstreamStatusBecomesInternal(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "bare unavailable", err: status.Error(codes.Unavailable, "connection refused")},
		{name: "bare internal", err: status.Error(codes.Internal, "boom")},
		{name: "wrapped unavailable", err: fmt.Errorf("get account %s: get account: %w", TEST_ACCOUNT_ID, status.Error(codes.Unavailable, "connection refused"))},
		{name: "wrapped internal", err: fmt.Errorf("get account %s: get account: %w", TEST_ACCOUNT_ID, status.Error(codes.Internal, "boom"))},
	}
	for _, rpc := range allRPCs() {
		for _, tt := range tests {
			t.Run(rpc.name+"/"+tt.name, func(t *testing.T) {
				fake := newFailingLedgerService(tt.err)
				srv, logBuf := newTestServer(fake)

				got, err := rpc.call(t.Context(), srv)

				require.ErrorIs(t, err, rpc.internalErr)
				assert.Equal(t, codes.Internal, status.Code(err))
				assert.Nil(t, got)
				assert.Contains(t, logBuf.String(), TEST_ERROR_LOG)
			})
		}
	}
}
