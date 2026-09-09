package repository

const TABLE_METADATA = "metadata"

const METADATA_COLUMNS = `
		id, user_id, type, status, object_path, filename,
		content_type, size_bytes, sha256, created_at, confirmed_at`

const (
	CREATE_METADATA_QUERY = `
		INSERT INTO ` + TABLE_METADATA + ` (
			id, user_id, type, object_path, filename
		)
		VALUES ($1, $2, $3::file_type, $4, $5)
		RETURNING ` + METADATA_COLUMNS

	GET_METADATA_QUERY = `
		SELECT ` + METADATA_COLUMNS + `
		FROM ` + TABLE_METADATA + `
		WHERE id = $1
		  AND user_id = $2`

	CONFIRM_METADATA_QUERY = `
		UPDATE ` + TABLE_METADATA + ` SET
			status       = 'confirmed'::file_status,
			content_type = $3,
			size_bytes   = $4,
			sha256       = $5,
			confirmed_at = now()
		WHERE id = $1
		  AND user_id = $2
		RETURNING ` + METADATA_COLUMNS

	LIST_METADATA_QUERY = `
		SELECT ` + METADATA_COLUMNS + `
		FROM ` + TABLE_METADATA + `
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`

	LIST_CONFIRMED_TYPES_QUERY = `
		SELECT DISTINCT type
		FROM ` + TABLE_METADATA + `
		WHERE user_id = $1
		  AND status = 'confirmed'::file_status`
)
