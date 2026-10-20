package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

func newCurrencies(t *testing.T, raw ...string) models.CurrencySet {
	t.Helper()
	currencies, err := models.NewCurrencySet(raw)
	require.NoError(t, err)
	return currencies
}

// TEST_ALL_CURRENCIES - набор валют по умолчанию
var TEST_ALL_CURRENCIES = []string{"RUB", "USD", "EUR"}

func boolPtr(value bool) *bool { return &value }

func TestRuleService_ListRules(t *testing.T) {
	twoRules := []models.RuleConfig{singleLimitRule(true), velocityRule(false)}
	tests := []struct {
		name    string
		rules   []models.RuleConfig
		listErr error
		want    []models.RuleConfig
		wantErr error
	}{
		{name: "two rules", rules: twoRules, want: twoRules},
		{name: "empty slice", rules: []models.RuleConfig{}, want: []models.RuleConfig{}},
		{name: "nil slice", rules: nil, want: nil},
		{name: "repository error", listErr: errDB, wantErr: errDB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{tt.rules}, listErr: tt.listErr}
			svc := NewRuleService(ruleRepo, newCurrencies(t, "RUB", "USD"))

			got, err := svc.ListRules(t.Context())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRuleService_UpdateRule(t *testing.T) {
	tests := []struct {
		name            string
		currencies      []string
		update          models.RuleUpdate
		repoResult      *models.RuleConfig
		repoErr         error
		wantErr         error
		wantRepoUpdates []models.RuleUpdate
	}{
		{
			name:            "enabled only without params check",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_VELOCITY, Enabled: boolPtr(false), OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_VELOCITY},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_VELOCITY, Enabled: boolPtr(false), OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "nothing set goes to repository",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_VELOCITY, OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_VELOCITY},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_VELOCITY, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "unknown code without params from repository",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: "unknown", Enabled: boolPtr(true), OfficerID: TEST_OFFICER_ID},
			repoErr:         domain.ErrRuleNotFound,
			wantErr:         domain.ErrRuleNotFound,
			wantRepoUpdates: []models.RuleUpdate{{Code: "unknown", Enabled: boolPtr(true), OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "repository error without params",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_VELOCITY, Enabled: boolPtr(true), OfficerID: TEST_OFFICER_ID},
			repoErr:         errDB,
			wantErr:         errDB,
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_VELOCITY, Enabled: boolPtr(true), OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:       "velocity zero max count",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_VELOCITY, Params: &models.RuleParams{MaxCount: 0, WindowSeconds: 60}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "velocity zero window",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_VELOCITY, Params: &models.RuleParams{MaxCount: 3, WindowSeconds: 0}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "velocity with limits",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_VELOCITY, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 100}, MaxCount: 3, WindowSeconds: 60}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "velocity empty params",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_VELOCITY, Params: &models.RuleParams{}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "unknown code with params",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: "unknown", Params: &models.RuleParams{}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrRuleNotFound,
		},
		{
			name:       "single limit zero threshold",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 0}}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "daily limit negative threshold",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: -1}}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "single limit with max count",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1000}, MaxCount: 3}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:       "same account with params",
			currencies: TEST_ALL_CURRENCIES,
			update:     models.RuleUpdate{Code: domain.RULE_SAME_ACCOUNT, Params: &models.RuleParams{MaxCount: 1}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:            "same account empty params",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_SAME_ACCOUNT, Params: &models.RuleParams{}, OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_SAME_ACCOUNT},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_SAME_ACCOUNT, Params: &models.RuleParams{}, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "single limit empty params",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{}, OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_SINGLE_AMOUNT_LIMIT},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{}, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "single limit valid params",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 5000}}, OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 5000}}, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "velocity minimal valid params with enabled",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_VELOCITY, Enabled: boolPtr(true), Params: &models.RuleParams{MaxCount: 1, WindowSeconds: 1}, OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_VELOCITY, Enabled: true},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_VELOCITY, Enabled: boolPtr(true), Params: &models.RuleParams{MaxCount: 1, WindowSeconds: 1}, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:            "repository error with valid params",
			currencies:      TEST_ALL_CURRENCIES,
			update:          models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 5000}}, OfficerID: TEST_OFFICER_ID},
			repoErr:         errDB,
			wantErr:         errDB,
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 5000}}, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:       "single limit currency outside set",
			currencies: []string{"RUB"},
			update:     models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_USD: 100}}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
		{
			name:            "single limit currency inside set",
			currencies:      []string{"RUB", "USD"},
			update:          models.RuleUpdate{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_USD: 100}}, OfficerID: TEST_OFFICER_ID},
			repoResult:      &models.RuleConfig{Code: domain.RULE_SINGLE_AMOUNT_LIMIT},
			wantRepoUpdates: []models.RuleUpdate{{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_USD: 100}}, OfficerID: TEST_OFFICER_ID}},
		},
		{
			name:       "daily limit currency outside set",
			currencies: []string{"RUB"},
			update:     models.RuleUpdate{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Params: &models.RuleParams{Limits: map[models.Currency]int64{TEST_USD: 100}}, OfficerID: TEST_OFFICER_ID},
			wantErr:    domain.ErrInvalidRuleParams,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleRepo := &fakeRuleRepo{updateResult: tt.repoResult, updateErr: tt.repoErr}
			svc := NewRuleService(ruleRepo, newCurrencies(t, tt.currencies...))

			got, err := svc.UpdateRule(t.Context(), tt.update)

			assert.Equal(t, tt.wantRepoUpdates, ruleRepo.updateCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tt.repoResult, got)
		})
	}
}
