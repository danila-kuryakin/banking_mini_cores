package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerRepo struct {
	db *pgxpool.Pool
}

func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{db: db}
}

// CreateCustomer заводит карточку и сразу первую запись истории.
func (r *CustomerRepo) CreateCustomer(ctx context.Context, userID uuid.UUID, status models.Status) (*models.Customer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create customer: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var resp models.Customer
	var statusRet string

	err = tx.QueryRow(ctx, CREATE_CUSTOMER_QUERY,
		userID,
		status,
	).Scan(
		&resp.ID,
		&resp.UserID,
		&statusRet,
		&resp.StatusChangedAt,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, domain.ErrCustomerExists
		}

		return nil, fmt.Errorf("create customer: %w", err)
	}

	resp.Status = models.Status(statusRet)

	_, err = tx.Exec(ctx, INSERT_INITIAL_STATUS_HISTORY_QUERY,
		userID,
		resp.Status,
		resp.StatusChangedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create customer: write status history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create customer: commit: %w", err)
	}

	return &resp, nil
}

// SetStatus переводит клиента в новый статус и пишет переход в историю транзакций.
func (r *CustomerRepo) SetStatus(
	ctx context.Context,
	userID uuid.UUID,
	to models.Status,
	reason string,
	actorID *uuid.UUID,
	allowedFrom []models.Status,
) (*models.StatusChange, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("set status: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	var previous string
	var changedAt time.Time

	err = tx.QueryRow(ctx, LOCK_CUSTOMER_STATUS_QUERY, userID).Scan(&previous, &changedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}

		return nil, fmt.Errorf("set status: lock customer: %w", err)
	}

	from := models.Status(previous)

	if from == to {
		return &models.StatusChange{Previous: from, Current: from, ChangedAt: changedAt}, nil
	}

	err = tx.QueryRow(ctx, SET_CUSTOMER_STATUS_QUERY, userID, to, statusStrings(allowedFrom)).Scan(&changedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStatusConflict
		}

		return nil, fmt.Errorf("set status: update customer: %w", err)
	}

	_, err = tx.Exec(ctx, INSERT_STATUS_HISTORY_QUERY, userID, from, to, reason, actorID, changedAt)
	if err != nil {
		return nil, fmt.Errorf("set status: write status history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("set status: commit: %w", err)
	}

	return &models.StatusChange{Previous: from, Current: to, ChangedAt: changedAt}, nil
}

// statusStrings переводит список статусов в срез строк: pgx кодирует его в
// массив customer_status, а именованный строковый тип для этого не берёт.
func statusStrings(in []models.Status) []string {
	out := make([]string, 0, len(in))

	for _, status := range in {
		out = append(out, string(status))
	}

	return out
}

func (r *CustomerRepo) GetCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error) {
	var resp models.Customer
	var status string

	err := r.db.QueryRow(ctx, GET_CUSTOMER_QUERY, userID).Scan(
		&resp.ID,
		&resp.UserID,
		&status,
		&resp.StatusChangedAt,
		&resp.Profile.FirstName,
		&resp.Profile.LastName,
		&resp.Profile.BirthDate,
		&resp.Profile.Citizenship,
		&resp.Profile.Phone,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}

	resp.Status = models.Status(status)

	return &resp, nil
}

func (r *CustomerRepo) GetStatusByUserID(ctx context.Context, userID uuid.UUID) (*models.Status, *time.Time, error) {
	var status string
	var changedAt time.Time

	err := r.db.QueryRow(ctx, GET_CUSTOMER_STATUS_BY_USER_ID_QUERY, userID).Scan(
		&status,
		&changedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domain.ErrCustomerNotFound
		}

		return nil, nil, fmt.Errorf("get customer status by user id: %w", err)
	}

	statusRet := models.Status(status)

	return &statusRet, &changedAt, nil
}

// UpdateProfile сохраняет анкету. Статус меняется вместе с ней (new ->
// profile_filled на полной анкете), поэтому запись анкеты и запись перехода в
// историю идут одной транзакцией.
func (r *CustomerRepo) UpdateProfile(ctx context.Context, userID uuid.UUID, profile models.Profile, status models.Status) (*models.Customer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("update profile: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	var previous string
	var previousChangedAt time.Time

	err = tx.QueryRow(ctx, LOCK_CUSTOMER_STATUS_QUERY, userID).Scan(&previous, &previousChangedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}

		return nil, fmt.Errorf("update profile: lock customer: %w", err)
	}

	var resp models.Customer
	var statusResp string

	err = tx.QueryRow(ctx, UPDATE_PROFILE_QUERY,
		userID,
		profile.FirstName,
		profile.LastName,
		profile.BirthDate,
		profile.Citizenship,
		profile.Phone,
		status,
	).Scan(
		&resp.ID,
		&resp.UserID,
		&statusResp,
		&resp.StatusChangedAt,
		&resp.Profile.FirstName,
		&resp.Profile.LastName,
		&resp.Profile.BirthDate,
		&resp.Profile.Citizenship,
		&resp.Profile.Phone,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProfileLocked
		}

		return nil, fmt.Errorf("update profile: %w", err)
	}
	resp.Status = models.Status(statusResp)

	if from := models.Status(previous); from != resp.Status {
		_, err = tx.Exec(ctx, INSERT_STATUS_HISTORY_QUERY, userID, from, resp.Status, "", nil, resp.StatusChangedAt)
		if err != nil {
			return nil, fmt.Errorf("update profile: write status history: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("update profile: commit: %w", err)
	}

	return &resp, nil
}

func (r *CustomerRepo) ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	rows, err := r.db.Query(ctx, LIST_CUSTOMERS_QUERY, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query customers list: %w", err)
	}
	defer rows.Close()

	customers := make([]*models.Customer, 0, limit)

	for rows.Next() {
		var ret models.Customer
		var status string

		err := rows.Scan(
			&ret.ID,
			&ret.UserID,
			&status,
			&ret.StatusChangedAt,
			&ret.Profile.FirstName,
			&ret.Profile.LastName,
			&ret.Profile.BirthDate,
			&ret.Profile.Citizenship,
			&ret.Profile.Phone,
			&ret.CreatedAt,
			&ret.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan customers list: %w", err)
		}
		ret.Status = models.Status(status)

		customers = append(customers, &ret)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customers list: %w", err)
	}

	return customers, nil
}
