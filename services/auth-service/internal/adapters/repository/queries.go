package repository

const (
	TABLE_USERS          = "users"
	TABLE_REFRESH_TOKENS = "refresh_tokens"
	TABLE_LOGIN_ATTEMPTS = "login_attempts"
)

const (
	USER_COLUMNS          = `id, email, password_hash, role, created_at, updated_at`
	REFRESH_TOKEN_COLUMNS = `id, user_id, token_hash, expires_at, revoked_at, created_at`
)

const (
	CREATE_USER_QUERY = `
		INSERT INTO ` + TABLE_USERS + ` (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING ` + USER_COLUMNS

	DELETE_USER_QUERY = `
		DELETE FROM ` + TABLE_USERS + `
		WHERE id = $1`

	GET_USER_BY_EMAIL_QUERY = `
		SELECT ` + USER_COLUMNS + `
		FROM ` + TABLE_USERS + `
		WHERE lower(email) = lower($1)`

	GET_USER_BY_ID_QUERY = `
		SELECT ` + USER_COLUMNS + `
		FROM ` + TABLE_USERS + `
		WHERE id = $1`
)

const (
	CREATE_REFRESH_TOKEN_QUERY = `
		INSERT INTO ` + TABLE_REFRESH_TOKENS + ` (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	GET_REFRESH_TOKEN_QUERY = `
		SELECT ` + REFRESH_TOKEN_COLUMNS + `
		FROM ` + TABLE_REFRESH_TOKENS + `
		WHERE token_hash = $1`

	REVOKE_REFRESH_TOKEN_QUERY = `
		UPDATE ` + TABLE_REFRESH_TOKENS + `
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`

	REVOKE_USER_TOKENS_QUERY = `
		UPDATE ` + TABLE_REFRESH_TOKENS + `
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`
)

const (
	GET_LOGIN_LOCKED_UNTIL_QUERY = `
		SELECT locked_until
		FROM ` + TABLE_LOGIN_ATTEMPTS + `
		WHERE email = $1 AND locked_until > $2`

	REGISTER_LOGIN_FAILURE_QUERY = `
		INSERT INTO ` + TABLE_LOGIN_ATTEMPTS + ` AS la (email, failed_count, window_started_at, locked_until, updated_at)
		VALUES ($1, 1, $2::timestamptz, CASE WHEN 1 >= $4::int THEN $5::timestamptz END, $2::timestamptz)
		ON CONFLICT (email) DO UPDATE SET
			failed_count = CASE
				WHEN la.window_started_at < $3::timestamptz
					OR (la.locked_until IS NOT NULL AND la.locked_until <= $2::timestamptz) THEN 1
				ELSE la.failed_count + 1
			END,
			window_started_at = CASE
				WHEN la.window_started_at < $3::timestamptz
					OR (la.locked_until IS NOT NULL AND la.locked_until <= $2::timestamptz) THEN EXCLUDED.window_started_at
				ELSE la.window_started_at
			END,
			locked_until = CASE
				WHEN la.window_started_at < $3::timestamptz
					OR (la.locked_until IS NOT NULL AND la.locked_until <= $2::timestamptz) THEN EXCLUDED.locked_until
				WHEN la.failed_count + 1 >= $4::int THEN $5::timestamptz
				ELSE la.locked_until
			END,
			updated_at = EXCLUDED.updated_at`

	RESET_LOGIN_ATTEMPTS_QUERY = `
		DELETE FROM ` + TABLE_LOGIN_ATTEMPTS + `
		WHERE email = $1`
)
