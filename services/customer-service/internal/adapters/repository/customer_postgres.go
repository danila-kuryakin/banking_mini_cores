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

func (r *CustomerRepo) CreateCustomer(ctx context.Context, userID uuid.UUID, status models.Status) (*models.Customer, error) {
	var resp models.Customer
	var statusRet string

	err := r.db.QueryRow(ctx, CREATE_CUSTOMER_QUERY,
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

	return &resp, nil
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

func (r *CustomerRepo) UpdateProfile(ctx context.Context, userID uuid.UUID, profile models.Profile, status models.Status) (*models.Customer, error) {

	var resp models.Customer
	var statusResp string

	err := r.db.QueryRow(ctx, UPDATE_PROFILE_QUERY,
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
		// Существование клиента сервис проверил перед вызовом, поэтому пустой
		// результат означает, что между проверкой и обновлением статус успел
		// уехать из "new" - профиль больше не редактируется.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProfileLocked
		}

		return nil, fmt.Errorf("update profile: %w", err)
	}
	resp.Status = models.Status(statusResp)

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
