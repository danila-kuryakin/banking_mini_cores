package repository

const ACCOUNT_COLUMNS = `id, user_id, number, currency, status, status_changed_at, created_at, updated_at`

const NEXT_SERIAL_QUERY = `SELECT nextval('account_number_seq')`

// Блокируем следующую запись.
const LOCK_USER_ACCOUNTS_QUERY = `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`

const COUNT_OPEN_ACCOUNTS_QUERY = `
	SELECT count(*)
	FROM accounts
	WHERE user_id = $1 AND currency = $2 AND status <> 'closed'`

const CREATE_ACCOUNT_QUERY = `
	INSERT INTO accounts (id, user_id, number, currency, status)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING ` + ACCOUNT_COLUMNS

const GET_ACCOUNT_QUERY = `
	SELECT ` + ACCOUNT_COLUMNS + `
	FROM accounts
	WHERE ($1::uuid IS NULL OR user_id = $1) AND id = $2`

const GET_ACCOUNT_BY_NUMBER_QUERY = `
	SELECT ` + ACCOUNT_COLUMNS + `
	FROM accounts
	WHERE number = $1`

const LIST_ACCOUNTS_QUERY = `
	SELECT ` + ACCOUNT_COLUMNS + `
	FROM accounts
	WHERE user_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2 OFFSET $3`

// Блокируем запись, которая уже существует, для обновления.
const LOCK_ACCOUNT_QUERY = `SELECT status FROM accounts WHERE id = $1 FOR UPDATE`

const TRANSITION_ACCOUNT_QUERY = `
	UPDATE accounts
	SET status = $2, status_changed_at = now(), updated_at = now()
	WHERE id = $1 AND status = ANY($3::account_status[])
	RETURNING ` + ACCOUNT_COLUMNS

const INSERT_HISTORY_QUERY = `
	INSERT INTO account_status_history (account_id, from_status, to_status, actor_id, reason, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)`

const LIST_HISTORY_QUERY = `
	SELECT id, account_id, from_status, to_status, actor_id, reason, created_at
	FROM account_status_history
	WHERE account_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2 OFFSET $3`
