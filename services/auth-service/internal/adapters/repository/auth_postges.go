package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepo struct {
	db *pgxpool.Pool
}

func NewAuthRepo(db *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{db: db}
}

func (r *AuthRepo) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	var created models.User

	err := r.db.QueryRow(ctx, CREATE_USER_QUERY, user.Email, user.PasswordHash, string(user.Role)).Scan(
		&created.ID,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, domain.ErrEmailTaken
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return &created, nil
}

func (r *AuthRepo) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.Exec(ctx, DELETE_USER_QUERY, id); err != nil {
		return fmt.Errorf("delete user %s: %w", id, err)
	}

	return nil
}

func (r *AuthRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	err := r.db.QueryRow(ctx, GET_USER_BY_EMAIL_QUERY, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *AuthRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User

	err := r.db.QueryRow(ctx, GET_USER_BY_ID_QUERY, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *AuthRepo) AddRefreshToken(ctx context.Context, token models.RefreshToken) error {
	_, err := r.db.Exec(ctx, CREATE_REFRESH_TOKEN_QUERY, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *AuthRepo) GetRefreshToken(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
	var token models.RefreshToken

	err := r.db.QueryRow(ctx, GET_REFRESH_TOKEN_QUERY, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	return &token, nil
}

func (r *AuthRepo) RevokeRefreshToken(ctx context.Context, hash []byte) error {
	if _, err := r.db.Exec(ctx, REVOKE_REFRESH_TOKEN_QUERY, hash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (r *AuthRepo) RevokeUserTokens(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, REVOKE_USER_TOKENS_QUERY, userID); err != nil {
		return fmt.Errorf("revoke tokens of user %s: %w", userID, err)
	}

	return nil
}

func (r *AuthRepo) RotateRefreshToken(ctx context.Context, oldHash []byte, next models.RefreshToken) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rotate refresh token: %w", err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			return
		}
	}(tx, ctx)

	revoked, err := tx.Exec(ctx, REVOKE_REFRESH_TOKEN_QUERY, oldHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if revoked.RowsAffected() == 0 {
		return domain.ErrRefreshTokenNotFound
	}

	_, err = tx.Exec(ctx, CREATE_REFRESH_TOKEN_QUERY, next.UserID, next.TokenHash, next.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rotate refresh token: %w", err)
	}

	return nil
}
