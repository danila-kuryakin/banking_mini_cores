//go:build !gremlins

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

// BUG: коды сработавших правил не должны попадать в decline_reason (R7)
func TestLedgerService_Transfer_DenyReasonWithoutRuleCodes(t *testing.T) {
	tests := []struct {
		name       string
		verdict    models.Verdict
		wantReason string
	}{
		{
			name:       "one_rule_with_check_id",
			verdict:    models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID, Reasons: []string{"velocity"}},
			wantReason: "denied by antifraud (check chk-1)",
		},
		{
			name:       "two_rules_with_check_id",
			verdict:    models.Verdict{Allowed: false, CheckID: TEST_CHECK_ID, Reasons: []string{"velocity", "daily_amount_limit"}},
			wantReason: "denied by antifraud (check chk-1)",
		},
		{
			name:       "one_rule_without_check_id",
			verdict:    models.Verdict{Allowed: false, Reasons: []string{"velocity"}},
			wantReason: "denied by antifraud",
		},
		{
			name:       "two_rules_without_check_id",
			verdict:    models.Verdict{Allowed: false, Reasons: []string{"velocity", "daily_amount_limit"}},
			wantReason: "denied by antifraud",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := defaultRepo()
			fraud := &fakeFraudChecker{verdict: tt.verdict}
			svc := newTestService(t, repo, defaultAccounts(), fraud, testCash())

			got, err := svc.Transfer(t.Context(), transferCmd("RUB"))

			assert.Equal(t, []declineCall{{Posting: transferPosting(TEST_TRANSFER_SUM), Reason: tt.wantReason}}, repo.declineCalls)
			assert.Empty(t, repo.postCalls)
			assert.Equal(t, declinedTx(), got)
			require.ErrorIs(t, err, domain.ErrTransferDenied)
		})
	}
}
