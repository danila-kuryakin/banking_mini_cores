package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type ApplicationRepo struct {
	db *pgxpool.Pool
}

func NewApplicationRepo(db *pgxpool.Pool) *ApplicationRepo {
	return &ApplicationRepo{db: db}
}

// Create заводит заявку и первую запись журнал событий одной транзакцией.
func (r *ApplicationRepo) Create(ctx context.Context, app models.Application) (*models.Application, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create application: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, CREATE_APPLICATION_QUERY,
		app.UserID,
		models.STATUS_SUBMITTED,
		app.FirstName,
		app.LastName,
		app.BirthDate,
	)

	created, err := scanApplication(row)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, domain.ErrActiveApplicationExists
		}

		return nil, fmt.Errorf("create application: %w", err)
	}

	_, err = tx.Exec(ctx, INSERT_INITIAL_EVENT_QUERY,
		created.ID,
		created.Status,
		"",
		created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create application: write event: %w", err)
	}

	if err := writeOutbox(ctx, tx, created, nil); err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create application: commit: %w", err)
	}

	return created, nil
}

// GetByID Возвращает заявки по ID.
func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Application, error) {
	app, err := scanApplication(r.db.QueryRow(ctx, GET_APPLICATION_QUERY, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrApplicationNotFound
		}

		return nil, fmt.Errorf("get application: %w", err)
	}

	return r.withHistory(ctx, app)
}

// GetLatestByUserID Возвращает заявки по UserID.
func (r *ApplicationRepo) GetLatestByUserID(ctx context.Context, userID uuid.UUID) (*models.Application, error) {
	app, err := scanApplication(r.db.QueryRow(ctx, GET_LATEST_APPLICATION_QUERY, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrApplicationNotFound
		}

		return nil, fmt.Errorf("get latest application: %w", err)
	}

	return r.withHistory(ctx, app)
}

// withHistory читает историю авто проверок и журнала событий.
func (r *ApplicationRepo) withHistory(ctx context.Context, app *models.Application) (*models.Application, error) {
	checks, err := listChecks(ctx, r.db, app.ID)
	if err != nil {
		return nil, err
	}

	events, err := listEvents(ctx, r.db, app.ID)
	if err != nil {
		return nil, err
	}

	app.Checks = checks
	app.Events = events

	return app, nil
}

// List Возвращает заявки после фильтрации, `ReviewerID` и пагинации, `Limit` и `Offset`.
func (r *ApplicationRepo) List(ctx context.Context, filter models.ApplicationFilter) ([]*models.Application, error) {
	rows, err := r.db.Query(ctx, LIST_APPLICATIONS_QUERY,
		statusStrings(filter.Statuses),
		filter.ReviewerID,
		filter.Limit,
		filter.Offset,
		filter.Unassigned,
	)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	return collectApplications(rows, filter.Limit)
}

// Transition переводит заявку в новый статус и пишет запись журнал событий одной транзакцией.
func (r *ApplicationRepo) Transition(
	ctx context.Context,
	id uuid.UUID,
	t models.Transition,
	allowedFrom []models.ApplicationStatus,
) (*models.Application, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("transition application: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	var previous string

	err = tx.QueryRow(ctx, LOCK_APPLICATION_QUERY, id).Scan(&previous)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrApplicationNotFound
		}

		return nil, fmt.Errorf("transition application: lock: %w", err)
	}

	from := models.ApplicationStatus(previous)

	row := tx.QueryRow(ctx, TRANSITION_APPLICATION_QUERY,
		id,
		t.To,
		t.Reason,
		documentTypeStrings(t.RequestedTypes),
		t.Decided,
		statusStrings(allowedFrom),
	)

	updated, err := scanApplication(row)
	if err != nil {
		// Заявка заблокирована выше и существует, поэтому пустой результат
		// означает одно: переход из текущего статуса не разрешён.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStatusConflict
		}

		return nil, fmt.Errorf("transition application: %w", err)
	}

	_, err = tx.Exec(ctx, INSERT_EVENT_QUERY, id, from, t.To, t.ActorID, t.Comment)
	if err != nil {
		return nil, fmt.Errorf("transition application: write event: %w", err)
	}

	if err := writeOutbox(ctx, tx, updated, t.ActorID); err != nil {
		return nil, fmt.Errorf("transition application: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("transition application: commit: %w", err)
	}

	return updated, nil
}

// TakeForAutoChecks забирает пачку поданных заявок и сразу переводит их в
// auto_checks, чтобы соседний воркер не взял те же самые.
func (r *ApplicationRepo) TakeForAutoChecks(ctx context.Context, limit int) ([]*models.Application, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("take for auto checks: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	allowedFrom := models.AllowedFrom(models.STATUS_AUTO_CHECKS)

	rows, err := tx.Query(ctx, TAKE_FOR_AUTO_CHECKS_QUERY, limit, statusStrings(allowedFrom))
	if err != nil {
		return nil, fmt.Errorf("take for auto checks: %w", err)
	}

	taken, err := collectApplications(rows, limit)
	rows.Close()

	if err != nil {
		return nil, err
	}

	for _, app := range taken {
		_, err = tx.Exec(ctx, INSERT_EVENT_QUERY,
			app.ID,
			models.STATUS_SUBMITTED,
			models.STATUS_AUTO_CHECKS,
			nil,
			"",
		)
		if err != nil {
			return nil, fmt.Errorf("take for auto checks: write event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("take for auto checks: commit: %w", err)
	}

	return taken, nil
}

// writeOutbox кладёт событие о новом статусе в outbox той же транзакцией, что и смену статуса.
func writeOutbox(ctx context.Context, tx pgx.Tx, app *models.Application, actorID *uuid.UUID) error {
	msg, ok, err := models.NewApplicationEvent(app, actorID)
	if err != nil {
		return fmt.Errorf("build event: %w", err)
	}

	if !ok {
		return nil
	}

	_, err = tx.Exec(ctx, INSERT_OUTBOX_QUERY, msg.ID, msg.Key, msg.EventType, []byte(msg.Payload), msg.OccurredAt)
	if err != nil {
		return fmt.Errorf("write outbox: %w", err)
	}

	return nil
}

// Claim закрепляет заявку за офицером. Статус при этом не меняется, поэтому и
// записи аудита здесь нет.
func (r *ApplicationRepo) Claim(ctx context.Context, id, officerID uuid.UUID) (*models.Application, error) {
	app, err := scanApplication(r.db.QueryRow(ctx, CLAIM_APPLICATION_QUERY, id, officerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAlreadyClaimed
		}

		return nil, fmt.Errorf("claim application: %w", err)
	}

	return app, nil
}
