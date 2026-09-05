package repository

const (
	TABLE_USERS          = "users"
	TABLE_REFRESH_TOKENS = "refresh_tokens"
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
