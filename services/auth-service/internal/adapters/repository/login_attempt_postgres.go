package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
)

type LoginAttemptRepo struct {
	db *pgxpool.Pool
}

func NewLoginAttemptRepo(db *pgxpool.Pool) *LoginAttemptRepo {
	return &LoginAttemptRepo{db: db}
}

func (r *LoginAttemptRepo) LockedUntil(ctx context.Context, email string, now time.Time) (*time.Time, error) {
	var lockedUntil time.Time

	err := r.db.QueryRow(ctx, GET_LOGIN_LOCKED_UNTIL_QUERY, email, now).Scan(&lockedUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get login lock: %w", err)
	}

	return &lockedUntil, nil
}

func (r *LoginAttemptRepo) RegisterFailure(ctx context.Context, email string, failure models.LoginFailure) error {
	_, err := r.db.Exec(ctx, REGISTER_LOGIN_FAILURE_QUERY,
		email,
		failure.At,
		failure.WindowStart,
		failure.MaxAttempts,
		failure.LockUntil,
	)
	if err != nil {
		return fmt.Errorf("register login failure: %w", err)
	}

	return nil
}

func (r *LoginAttemptRepo) Reset(ctx context.Context, email string) error {
	if _, err := r.db.Exec(ctx, RESET_LOGIN_ATTEMPTS_QUERY, email); err != nil {
		return fmt.Errorf("reset login attempts: %w", err)
	}

	return nil
}
