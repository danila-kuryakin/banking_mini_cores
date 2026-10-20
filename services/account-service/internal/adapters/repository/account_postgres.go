package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

type AccountRepo struct {
	db *pgxpool.Pool
}

func NewAccountRepo(db *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{db: db}
}

func (r *AccountRepo) NextSerial(ctx context.Context) (int64, error) {
	var serial int64

	if err := r.db.QueryRow(ctx, NEXT_SERIAL_QUERY).Scan(&serial); err != nil {
		return 0, fmt.Errorf("next account serial: %w", err)
	}

	return serial, nil
}

// Create проверяет лимит и заводит счёт под advisory-блокировкой клиента, иначе два запроса пройдут лимит вместе.
func (r *AccountRepo) Create(ctx context.Context, account models.Account, maxPerCurrency int) (*models.Account, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create account: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, LOCK_USER_ACCOUNTS_QUERY, account.UserID.String()); err != nil {
		return nil, fmt.Errorf("create account: lock user: %w", err)
	}

	var count int

	err = tx.QueryRow(ctx, COUNT_OPEN_ACCOUNTS_QUERY, account.UserID, account.Currency).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("create account: count: %w", err)
	}

	if count >= maxPerCurrency {
		return nil, domain.ErrAccountLimitReached
	}

	created, err := scanAccount(tx.QueryRow(ctx, CREATE_ACCOUNT_QUERY,
		account.ID,
		account.UserID,
		account.Number,
		account.Currency,
		models.STATUS_ACTIVE,
	))
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	if err := insertHistory(ctx, tx, created.ID, nil, created.Status, nil, "", created); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create account: commit: %w", err)
	}

	return created, nil
}

func (r *AccountRepo) GetByID(ctx context.Context, userID *uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	account, err := scanAccount(r.db.QueryRow(ctx, GET_ACCOUNT_QUERY, userID, accountID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}

		return nil, fmt.Errorf("get account: %w", err)
	}

	return account, nil
}

func (r *AccountRepo) GetByNumber(ctx context.Context, number string) (*models.Account, error) {
	account, err := scanAccount(r.db.QueryRow(ctx, GET_ACCOUNT_BY_NUMBER_QUERY, number))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}

		return nil, fmt.Errorf("get account by number: %w", err)
	}

	return account, nil
}

func (r *AccountRepo) ListByUser(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Account, error) {
	rows, err := r.db.Query(ctx, LIST_ACCOUNTS_QUERY, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*models.Account, 0, page.Limit)

	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan accounts: %w", err)
		}

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}

	return accounts, nil
}

func (r *AccountRepo) ListHistory(ctx context.Context, accountID uuid.UUID, page models.Page) ([]*models.StatusChange, error) {
	rows, err := r.db.Query(ctx, LIST_HISTORY_QUERY, accountID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list status history: %w", err)
	}
	defer rows.Close()

	changes := make([]*models.StatusChange, 0, page.Limit)

	for rows.Next() {
		var (
			change models.StatusChange
			from   *string
			to     string
		)

		err := rows.Scan(
			&change.ID,
			&change.AccountID,
			&from,
			&to,
			&change.ActorID,
			&change.Reason,
			&change.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan status history: %w", err)
		}

		if from != nil {
			change.From = models.AccountStatus(*from)
		}

		change.To = models.AccountStatus(to)
		changes = append(changes, &change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status history: %w", err)
	}

	return changes, nil
}

func (r *AccountRepo) Transition(
	ctx context.Context,
	accountID uuid.UUID,
	t models.Transition,
	allowedFrom []models.AccountStatus,
) (*models.Account, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("transition account: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	var previous string

	if err := tx.QueryRow(ctx, LOCK_ACCOUNT_QUERY, accountID).Scan(&previous); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}

		return nil, fmt.Errorf("transition account: lock: %w", err)
	}

	from := models.AccountStatus(previous)

	updated, err := scanAccount(tx.QueryRow(ctx, TRANSITION_ACCOUNT_QUERY, accountID, t.To, statusStrings(allowedFrom)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStatusConflict
		}

		return nil, fmt.Errorf("transition account: %w", err)
	}

	if err := insertHistory(ctx, tx, accountID, &from, t.To, t.ActorID, t.Reason, updated); err != nil {
		return nil, fmt.Errorf("transition account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("transition account: commit: %w", err)
	}

	return updated, nil
}

// insertHistory пишет переход с тем же временем, что и status_changed_at счёта.
func insertHistory(
	ctx context.Context,
	tx pgx.Tx,
	accountID uuid.UUID,
	from *models.AccountStatus,
	to models.AccountStatus,
	actorID *uuid.UUID,
	reason string,
	account *models.Account,
) error {
	_, err := tx.Exec(ctx, INSERT_HISTORY_QUERY, accountID, from, to, actorID, reason, account.StatusChangedAt)
	if err != nil {
		return fmt.Errorf("write history: %w", err)
	}

	return nil
}

type row interface {
	Scan(dest ...any) error
}

func scanAccount(r row) (*models.Account, error) {
	var (
		account  models.Account
		currency string
		status   string
	)

	err := r.Scan(
		&account.ID,
		&account.UserID,
		&account.Number,
		&currency,
		&status,
		&account.StatusChangedAt,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	account.Currency = models.Currency(currency)
	account.Status = models.AccountStatus(status)

	return &account, nil
}

func statusStrings(statuses []models.AccountStatus) []string {
	out := make([]string, 0, len(statuses))

	for _, status := range statuses {
		out = append(out, string(status))
	}

	return out
}
