//go:build !gremlins

package server

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
)

// TestAccountServer_CurrencyNotAllowed проверяет, что наружу уходит ровно ErrCurrencyIsNotValid без ввода клиента.
func TestAccountServer_CurrencyNotAllowed(t *testing.T) {
	currencyErrors := []struct {
		name   string
		svcErr error
	}{
		{name: "bare", svcErr: domain.ErrCurrencyNotAllowed},
		{name: "with client input", svcErr: fmt.Errorf("%w: %s", domain.ErrCurrencyNotAllowed, "GEL")},
		{name: "wrapped", svcErr: fmt.Errorf("open account: %w", domain.ErrCurrencyNotAllowed)},
	}

	type currencyCase struct {
		name   string
		rpc    rpcCase
		svcErr error
	}
	var tests []currencyCase
	for _, rpc := range allRPCs() {
		for _, ce := range currencyErrors {
			tests = append(tests, currencyCase{name: rpc.name + "/" + ce.name, rpc: rpc, svcErr: ce.svcErr})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAccountService{}
			tt.rpc.setErr(fake, tt.svcErr)
			srv := newTestServer(fake)

			respIsNil, err := tt.rpc.call(t, srv)

			require.ErrorIs(t, err, domain.ErrCurrencyIsNotValid)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Equal(t, "currency is not allowed", status.Convert(err).Message())
			assert.True(t, respIsNil)
		})
	}
}
