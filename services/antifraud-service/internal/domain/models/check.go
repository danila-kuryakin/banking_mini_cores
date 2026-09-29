package models

import (
	"time"

	"github.com/google/uuid"
)

// Verdict совпадает с меткой enum af_verdict в Postgres.
type Verdict string

const (
	VERDICT_ALLOW Verdict = "allow"
	VERDICT_DENY  Verdict = "deny"
)

// TransferCheck - перевод, который ledger собирается провести.
type TransferCheck struct {
	InitiatorID   uuid.UUID
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Amount        int64
	Currency      Currency
	// Момент проверки: от него отсчитываются окно velocity и текущий день.
	At time.Time
}

// Hit - сработавшее правило и объяснение, почему.
type Hit struct {
	RuleCode string
	Reason   string
}

// Check - запись журнала af_checks.
type Check struct {
	ID       uuid.UUID
	Transfer TransferCheck
	Verdict  Verdict
	Hits     []Hit
}

// Reasons - объяснения отказа в виде "правило: причина".
func (c *Check) Reasons() []string {
	reasons := make([]string, 0, len(c.Hits))

	for _, hit := range c.Hits {
		reasons = append(reasons, hit.RuleCode+": "+hit.Reason)
	}

	return reasons
}

// VerdictFor - любой сработавший запрет даёт DENY.
func VerdictFor(hits []Hit) Verdict {
	if len(hits) > 0 {
		return VERDICT_DENY
	}

	return VERDICT_ALLOW
}
