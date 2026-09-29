package repository

const RULE_COLUMNS = `code, description, enabled, params, updated_by, updated_at`

const LIST_RULES_QUERY = `SELECT ` + RULE_COLUMNS + ` FROM af_rules ORDER BY code`

// UPDATE_RULE_QUERY: NULL в $2 или $3 оставляет поле как было.
const UPDATE_RULE_QUERY = `
UPDATE af_rules
SET enabled = COALESCE($2, enabled),
    params = COALESCE($3, params),
    updated_by = $4,
    updated_at = now()
WHERE code = $1
RETURNING ` + RULE_COLUMNS

// LOCK_ACCOUNT_CHECKS_QUERY выстраивает проверки одного счёта в очередь до коммита транзакции.
const LOCK_ACCOUNT_CHECKS_QUERY = `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`

const COUNT_ALLOWED_QUERY = `
SELECT count(*)
FROM af_checks
WHERE from_account_id = $1 AND verdict = 'allow' AND created_at > $2`

const DAILY_TOTAL_QUERY = `
SELECT coalesce(sum(amount), 0)
FROM af_daily_totals
WHERE account_id = $1 AND day = $2 AND currency = $3`

const INSERT_CHECK_QUERY = `
INSERT INTO af_checks (id, initiator_id, from_account_id, to_account_id, amount, currency, verdict, triggered_rules, reasons, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

const INSERT_PROCESSED_EVENT_QUERY = `
INSERT INTO processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING`

const ADD_DAILY_TOTAL_QUERY = `
INSERT INTO af_daily_totals (account_id, day, currency, amount)
VALUES ($1, $2, $3, $4)
ON CONFLICT (account_id, day, currency)
DO UPDATE SET amount = af_daily_totals.amount + EXCLUDED.amount, updated_at = now()`
