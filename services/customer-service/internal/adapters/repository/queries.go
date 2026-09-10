package repository

const (
	TABLE_CUSTOMERS      = "customers"
	TABLE_STATUS_HISTORY = "customer_status_history"
)

const CUSTOMER_COLUMNS = `
		id, user_id, status, status_changed_at,
		first_name, last_name, birth_date, citizenship, phone,
		created_at, updated_at`

const (
	CREATE_CUSTOMER_QUERY = `
		INSERT INTO ` + TABLE_CUSTOMERS + ` (
			user_id, status
		)
		VALUES ($1, $2)
		RETURNING id, user_id, status, status_changed_at, created_at, updated_at`

	GET_CUSTOMER_QUERY = `
		SELECT ` + CUSTOMER_COLUMNS + `
		FROM ` + TABLE_CUSTOMERS + `
		WHERE user_id = $1`

	UPDATE_PROFILE_QUERY = `
		UPDATE ` + TABLE_CUSTOMERS + ` SET
			first_name          = $2,
			last_name           = $3,
			birth_date          = $4,
			citizenship         = $5,
			phone               = $6,
			status              = $7::customer_status,
			status_changed_at   = CASE WHEN status <> $7::customer_status THEN now() ELSE status_changed_at END,
			updated_at          = now()
		WHERE user_id = $1
		  AND status = 'new'::customer_status
		RETURNING ` + CUSTOMER_COLUMNS

	LIST_CUSTOMERS_QUERY = `
		SELECT ` + CUSTOMER_COLUMNS + `
		FROM ` + TABLE_CUSTOMERS + `
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2`

	GET_CUSTOMER_STATUS_BY_USER_ID_QUERY = `
		SELECT status, status_changed_at
		FROM ` + TABLE_CUSTOMERS + `
		WHERE user_id = $1::uuid`

	LOCK_CUSTOMER_STATUS_QUERY = `
		SELECT status, status_changed_at
		FROM ` + TABLE_CUSTOMERS + `
		WHERE user_id = $1
		FOR UPDATE`

	SET_CUSTOMER_STATUS_QUERY = `
		UPDATE ` + TABLE_CUSTOMERS + ` SET
			status            = $2::customer_status,
			status_changed_at = now(),
			updated_at        = now()
		WHERE user_id = $1
		  AND status = ANY($3::customer_status[])
		RETURNING status_changed_at`

	INSERT_STATUS_HISTORY_QUERY = `
		INSERT INTO ` + TABLE_STATUS_HISTORY + ` (
			user_id, from_status, to_status, reason, actor_id, created_at
		)
		VALUES ($1, $2::customer_status, $3::customer_status, $4, $5, $6)`

	INSERT_INITIAL_STATUS_HISTORY_QUERY = `
		INSERT INTO ` + TABLE_STATUS_HISTORY + ` (
			user_id, from_status, to_status, created_at
		)
		VALUES ($1, NULL, $2::customer_status, $3)`
)
