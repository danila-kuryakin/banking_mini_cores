package rules

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// fakeStats - статистика счёта без базы.
type fakeStats struct {
	allowed    int
	dailyTotal int64
	since      time.Time
	day        time.Time
}

func (s *fakeStats) CountAllowed(_ context.Context, _ uuid.UUID, since time.Time) (int, error) {
	s.since = since

	return s.allowed, nil
}

func (s *fakeStats) DailyTotal(_ context.Context, _ uuid.UUID, _ models.Currency, day time.Time) (int64, error) {
	s.day = day

	return s.dailyTotal, nil
}

var testNow = time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC)

func testTransfer(amount int64) models.TransferCheck {
	return models.TransferCheck{
		InitiatorID:   uuid.New(),
		FromAccountID: uuid.New(),
		ToAccountID:   uuid.New(),
		Amount:        amount,
		Currency:      "RUB",
		At:            testNow,
	}
}

func TestSingleAmountLimit(t *testing.T) {
	rule := singleAmountLimit{limits: map[models.Currency]int64{"RUB": 1000}}

	cases := []struct {
		name     string
		transfer models.TransferCheck
		hit      bool
	}{
		{"below limit", testTransfer(999), false},
		{"exactly limit", testTransfer(1000), false},
		{"above limit", testTransfer(1001), true},
		{"currency without limit", func() models.TransferCheck { tr := testTransfer(1_000_000); tr.Currency = "USD"; return tr }(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason, err := rule.Check(context.Background(), tc.transfer, &fakeStats{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (reason != "") != tc.hit {
				t.Fatalf("hit = %v, reason %q", reason != "", reason)
			}
		})
	}
}

func TestDailyAmountLimit(t *testing.T) {
	rule := dailyAmountLimit{limits: map[models.Currency]int64{"RUB": 1000}}

	cases := []struct {
		name  string
		total int64
		sum   int64
		hit   bool
	}{
		{"fits", 400, 600, false},
		{"reaches limit exactly", 500, 500, false},
		{"exceeds limit", 500, 501, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stats := &fakeStats{dailyTotal: tc.total}

			reason, err := rule.Check(context.Background(), testTransfer(tc.sum), stats)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (reason != "") != tc.hit {
				t.Fatalf("hit = %v, reason %q", reason != "", reason)
			}

			if want := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC); !stats.day.Equal(want) {
				t.Fatalf("day = %v, want %v", stats.day, want)
			}
		})
	}
}

func TestVelocity(t *testing.T) {
	rule := newVelocity(models.RuleParams{MaxCount: 3, WindowSeconds: 60})

	cases := []struct {
		name    string
		allowed int
		hit     bool
	}{
		{"first transfer", 0, false},
		{"last allowed", 2, false},
		{"over the limit", 3, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stats := &fakeStats{allowed: tc.allowed}

			reason, err := rule.Check(context.Background(), testTransfer(100), stats)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (reason != "") != tc.hit {
				t.Fatalf("hit = %v, reason %q", reason != "", reason)
			}

			if want := testNow.Add(-time.Minute); !stats.since.Equal(want) {
				t.Fatalf("since = %v, want %v", stats.since, want)
			}
		})
	}
}

func TestSameAccount(t *testing.T) {
	transfer := testTransfer(100)

	if reason, _ := (sameAccount{}).Check(context.Background(), transfer, nil); reason != "" {
		t.Fatalf("different accounts must pass, got %q", reason)
	}

	transfer.ToAccountID = transfer.FromAccountID

	if reason, _ := (sameAccount{}).Check(context.Background(), transfer, nil); reason == "" {
		t.Fatalf("same account must be denied")
	}
}

func TestEvaluateCollectsAllHits(t *testing.T) {
	rules, err := Build([]models.RuleConfig{
		{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: models.RuleParams{Limits: map[models.Currency]int64{"RUB": 10}}},
		{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Enabled: true, Params: models.RuleParams{Limits: map[models.Currency]int64{"RUB": 10}}},
		{Code: domain.RULE_VELOCITY, Enabled: false, Params: models.RuleParams{MaxCount: 1, WindowSeconds: 60}},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	hits, err := Evaluate(context.Background(), rules, testTransfer(100), &fakeStats{allowed: 5})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if len(hits) != 2 {
		t.Fatalf("want 2 hits from enabled rules, got %v", hits)
	}

	if models.VerdictFor(hits) != models.VERDICT_DENY {
		t.Fatalf("hits must deny")
	}

	if models.VerdictFor(nil) != models.VERDICT_ALLOW {
		t.Fatalf("no hits must allow")
	}
}

func TestBuildUnknownRule(t *testing.T) {
	if _, err := Build([]models.RuleConfig{{Code: "unknown", Enabled: true}}); err == nil {
		t.Fatalf("unknown enabled rule must fail")
	}

	if _, err := Build([]models.RuleConfig{{Code: "unknown", Enabled: false}}); err != nil {
		t.Fatalf("disabled rule is not built: %v", err)
	}
}

func TestValidate(t *testing.T) {
	currencies, err := models.NewCurrencySet([]string{"RUB", "USD"})
	if err != nil {
		t.Fatalf("currencies: %v", err)
	}

	cases := []struct {
		name   string
		code   string
		params models.RuleParams
		want   error
	}{
		{"valid limits", domain.RULE_SINGLE_AMOUNT_LIMIT, models.RuleParams{Limits: map[models.Currency]int64{"RUB": 1}}, nil},
		{"empty limits", domain.RULE_DAILY_AMOUNT_LIMIT, models.RuleParams{}, nil},
		{"zero limit", domain.RULE_SINGLE_AMOUNT_LIMIT, models.RuleParams{Limits: map[models.Currency]int64{"RUB": 0}}, domain.ErrInvalidRuleParams},
		{"foreign currency", domain.RULE_DAILY_AMOUNT_LIMIT, models.RuleParams{Limits: map[models.Currency]int64{"GEL": 1}}, domain.ErrInvalidRuleParams},
		{"velocity on limit rule", domain.RULE_SINGLE_AMOUNT_LIMIT, models.RuleParams{MaxCount: 1}, domain.ErrInvalidRuleParams},
		{"valid velocity", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: 60}, nil},
		{"velocity without window", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5}, domain.ErrInvalidRuleParams},
		{"same account with params", domain.RULE_SAME_ACCOUNT, models.RuleParams{MaxCount: 1}, domain.ErrInvalidRuleParams},
		{"unknown rule", "unknown", models.RuleParams{}, domain.ErrRuleNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.code, tc.params, currencies)

			if tc.want == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}
