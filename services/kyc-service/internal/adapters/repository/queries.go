package repository

const (
	TABLE_APPLICATIONS = "applications"
	TABLE_CHECKS       = "kyc_checks"
	TABLE_EVENTS       = "kyc_events"
	TABLE_SANCTIONS    = "sanctions_list"
)

// Колонки заявки в порядке, который ждёт scanApplication.
const APPLICATION_COLUMNS = `
		id, user_id, status, reviewer_id, reason, requested_types,
		first_name, last_name, birth_date,
		submitted_at, decided_at, created_at, updated_at`

const (
	// Создание заявки.
	CREATE_APPLICATION_QUERY = `
		INSERT INTO ` + TABLE_APPLICATIONS + ` (
			user_id, status, first_name, last_name, birth_date
		)
		VALUES ($1, $2::application_status, $3, $4, $5)
		RETURNING ` + APPLICATION_COLUMNS

	// Отдаёт заявку по id.
	GET_APPLICATION_QUERY = `
		SELECT ` + APPLICATION_COLUMNS + `
		FROM ` + TABLE_APPLICATIONS + `
		WHERE id = $1`

	// Отдаёт последнюю заявку по user_id.
	GET_LATEST_APPLICATION_QUERY = `
		SELECT ` + APPLICATION_COLUMNS + `
		FROM ` + TABLE_APPLICATIONS + `
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1`

	// Возвращает очередь для офицера; пустые статусы и NULL в reviewer_id - без фильтра, $5 - только ничьи заявки.
	LIST_APPLICATIONS_QUERY = `
		SELECT ` + APPLICATION_COLUMNS + `
		FROM ` + TABLE_APPLICATIONS + `
		WHERE (cardinality($1::application_status[]) = 0 OR status = ANY($1::application_status[]))
		  AND ($2::uuid IS NULL OR reviewer_id = $2::uuid)
		  AND (NOT $5::boolean OR reviewer_id IS NULL)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4`

	// Читает статус заявки и блокирует строку до конца транзакции.
	LOCK_APPLICATION_QUERY = `
		SELECT status
		FROM ` + TABLE_APPLICATIONS + `
		WHERE id = $1
		FOR UPDATE`

	// Переводит заявку в новый статус, если текущий входит в допустимые переходы status.
	TRANSITION_APPLICATION_QUERY = `
		UPDATE ` + TABLE_APPLICATIONS + ` SET
			status          = $2::application_status,
			reason          = $3,
			requested_types = $4::document_type[],
			decided_at      = CASE WHEN $5 THEN now() ELSE decided_at END,
			updated_at      = now()
		WHERE id = $1
		  AND status = ANY($6::application_status[])
		RETURNING ` + APPLICATION_COLUMNS

	// Забирает пачку заявок для воркера и переводит в auto_checks; SKIP LOCKED пропускает чужие.
	TAKE_FOR_AUTO_CHECKS_QUERY = `
		WITH taken AS (
			SELECT id
			FROM ` + TABLE_APPLICATIONS + `
			WHERE status = ANY($2::application_status[])
			ORDER BY created_at, id
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE ` + TABLE_APPLICATIONS + ` a SET
			status     = 'auto_checks'::application_status,
			updated_at = now()
		FROM taken
		WHERE a.id = taken.id
		RETURNING ` + `a.id, a.user_id, a.status, a.reviewer_id, a.reason, a.requested_types,
			a.first_name, a.last_name, a.birth_date,
			a.submitted_at, a.decided_at, a.created_at, a.updated_at`

	// Закрепляет заявку за офицером; reviewer_id IS NULL - оптимистическая блокировка.
	CLAIM_APPLICATION_QUERY = `
		UPDATE ` + TABLE_APPLICATIONS + ` SET
			reviewer_id = $2,
			updated_at  = now()
		WHERE id = $1
		  AND status = 'in_review'::application_status
		  AND reviewer_id IS NULL
		RETURNING ` + APPLICATION_COLUMNS
)

const (
	// Пишет запись аудита о переходе статуса.
	INSERT_EVENT_QUERY = `
		INSERT INTO ` + TABLE_EVENTS + ` (
			application_id, from_status, to_status, actor_id, comment
		)
		VALUES ($1, $2::application_status, $3::application_status, $4, $5)`

	// Пишет первую запись аудита при создании заявки: исходного статуса нет.
	INSERT_INITIAL_EVENT_QUERY = `
		INSERT INTO ` + TABLE_EVENTS + ` (
			application_id, from_status, to_status, comment, created_at
		)
		VALUES ($1, NULL, $2::application_status, $3, $4)`

	// Отдаёт историю переходов заявки от создания к последнему.
	LIST_EVENTS_QUERY = `
		SELECT id, application_id, from_status, to_status, actor_id, comment, created_at
		FROM ` + TABLE_EVENTS + `
		WHERE application_id = $1
		ORDER BY created_at, id`
)

const (
	// Пишет результат проверки; повтор того же типа по заявке перезаписывает строку.
	SAVE_CHECK_QUERY = `
		INSERT INTO ` + TABLE_CHECKS + ` (
			application_id, type, result, details
		)
		VALUES ($1, $2::check_type, $3::check_result, $4)
		ON CONFLICT (application_id, type) DO UPDATE SET
			result     = EXCLUDED.result,
			details    = EXCLUDED.details,
			created_at = now()
		RETURNING id, application_id, type, result, details, created_at`

	// Отдаёт результаты проверок заявки.
	LIST_CHECKS_QUERY = `
		SELECT id, application_id, type, result, details, created_at
		FROM ` + TABLE_CHECKS + `
		WHERE application_id = $1
		ORDER BY created_at, id`
)

// Ищет человека в санкционном списке без учёта регистра.
const FIND_SANCTION_QUERY = `
		SELECT id, first_name, last_name, birth_date, source
		FROM ` + TABLE_SANCTIONS + `
		WHERE lower(last_name) = lower($1)
		  AND lower(first_name) = lower($2)
		  AND birth_date = $3
		LIMIT 1`

// Добавляет человека в санкционный список; дубль падает на уникальном индексе без учёта регистра.
const INSERT_SANCTION_QUERY = `
		INSERT INTO ` + TABLE_SANCTIONS + ` (
			first_name, last_name, birth_date, source
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, first_name, last_name, birth_date, source`

// Отдаёт санкционный список по алфавиту; сортировка без учёта регистра, как и сверка.
const LIST_SANCTIONS_QUERY = `
		SELECT id, first_name, last_name, birth_date, source
		FROM ` + TABLE_SANCTIONS + `
		ORDER BY lower(last_name), lower(first_name), birth_date
		LIMIT $1 OFFSET $2`
