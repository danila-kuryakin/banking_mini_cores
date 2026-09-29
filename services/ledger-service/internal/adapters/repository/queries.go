package repository

const ENSURE_CASH_ACCOUNT_QUERY = `
INSERT INTO balances (account_id, kind, currency)
VALUES ($1, 'cash', $2)
ON CONFLICT (currency) WHERE kind = 'cash' DO NOTHING`

const LIST_CASH_ACCOUNTS_QUERY = `SELECT currency, account_id FROM balances WHERE kind = 'cash'`

// ENSURE_BALANCE_QUERY заводит строку клиента при первом движении; существующая строка не трогается.
const ENSURE_BALANCE_QUERY = `
INSERT INTO balances (account_id, currency)
VALUES ($1, $2)
ON CONFLICT (account_id) DO NOTHING`

// LOCK_BALANCES_QUERY блокирует строки в порядке account_id.
const LOCK_BALANCES_QUERY = `
SELECT account_id, kind, currency, amount
FROM balances
WHERE account_id = ANY($1::uuid[])
ORDER BY account_id
FOR UPDATE`

const UPDATE_BALANCE_QUERY = `
UPDATE balances
SET amount = amount + $2, updated_at = now()
WHERE account_id = $1`

const TRANSACTION_COLUMNS = `id, type, status, from_account_id, to_account_id, amount, currency, initiator_id, decline_reason, created_at`

const INSERT_TRANSACTION_QUERY = `
INSERT INTO transactions (id, type, status, from_account_id, to_account_id, amount, currency, initiator_id, decline_reason)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING ` + TRANSACTION_COLUMNS

const INSERT_ENTRY_QUERY = `
INSERT INTO entries (id, transaction_id, account_id, direction, amount, currency, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

const INSERT_OUTBOX_QUERY = `
INSERT INTO outbox (id, message_key, event_type, payload, occurred_at)
VALUES ($1, $2, $3, $4, $5)`

const GET_TRANSACTION_QUERY = `SELECT ` + TRANSACTION_COLUMNS + ` FROM transactions WHERE id = $1`

const LIST_ENTRIES_QUERY = `
SELECT id, transaction_id, account_id, direction, amount, currency, created_at
FROM entries
WHERE transaction_id = $1
ORDER BY direction, id`

// LIST_TRANSACTIONS_QUERY - выписка: все исходящие и только проведённые входящие.
const LIST_TRANSACTIONS_QUERY = `
SELECT ` + TRANSACTION_COLUMNS + `
FROM transactions
WHERE from_account_id = $1 OR (to_account_id = $1 AND status = 'completed')
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3`

const GET_BALANCE_QUERY = `SELECT account_id, currency, amount FROM balances WHERE account_id = $1`

// TAKE_OUTBOX_QUERY с SKIP LOCKED: вторая реплика возьмёт другую пачку, а не будет ждать первую.
const TAKE_OUTBOX_QUERY = `
SELECT id, message_key, event_type, payload, occurred_at
FROM outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED`

const MARK_OUTBOX_PUBLISHED_QUERY = `UPDATE outbox SET published_at = now() WHERE id = ANY($1::uuid[])`
