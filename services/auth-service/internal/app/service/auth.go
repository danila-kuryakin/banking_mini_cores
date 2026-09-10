package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/password"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
)

type AuthService struct {
	repo      *repository.Repository
	tokens    *token.Manager
	customers interfaces.CustomerProfiles
}

func NewAuthService(
	repo *repository.Repository,
	tokens *token.Manager,
	customers interfaces.CustomerProfiles,
) *AuthService {
	return &AuthService{
		repo:      repo,
		tokens:    tokens,
		customers: customers,
	}
}

// Register заводит учётную запись и сразу карточку клиента в customer-service.
func (s *AuthService) Register(ctx context.Context, email, plainPassword string) (*models.User, error) {
	user, err := s.createUser(ctx, email, plainPassword, domain.ROLE_CLIENT)
	if err != nil {
		return nil, err
	}

	err = s.customers.CreateProfile(ctx, user.ID)
	if err != nil && !errors.Is(err, domain.ErrProfileAlreadyExists) {
		if delErr := s.repo.Auth.DeleteUser(ctx, user.ID); delErr != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrProfileNotCreated, errors.Join(err, delErr))
		}

		return nil, fmt.Errorf("%w: %w", domain.ErrProfileNotCreated, err)
	}

	return user, nil
}

func (s *AuthService) CreateOfficer(ctx context.Context, email, plainPassword string) (*models.User, error) {
	return s.createUser(ctx, email, plainPassword, domain.ROLE_OFFICER)
}

func (s *AuthService) EnsureAdmin(ctx context.Context, email, plainPassword string) (bool, error) {
	normalizedEmail, _, err := normalizeCredentials(email, plainPassword)
	if err != nil {
		return false, err
	}

	_, err = s.repo.Auth.GetUserByEmail(ctx, normalizedEmail)
	if err == nil {
		return false, nil
	}

	if !errors.Is(err, domain.ErrUserNotFound) {
		return false, err
	}

	if _, err := s.createUser(ctx, email, plainPassword, domain.ROLE_ADMIN); err != nil {
		return false, err
	}

	return true, nil
}

func (s *AuthService) Login(ctx context.Context, email, plainPassword string) (*models.User, *token.Pair, error) {
	email, plainPassword, err := normalizeCredentials(email, plainPassword)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.repo.Auth.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if err := password.Verify(user.PasswordHash, plainPassword); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	pair, err := s.tokens.NewPair(user)
	if err != nil {
		return nil, nil, err
	}

	stored := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: token.Hash(pair.Refresh),
		ExpiresAt: pair.RefreshExpiresAt,
	}

	if err := s.repo.Auth.AddRefreshToken(ctx, stored); err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*token.Pair, error) {
	if refreshToken == "" {
		return nil, domain.ErrInvalidRefreshToken
	}

	hash := token.Hash(refreshToken)

	refToken, err := s.repo.Auth.GetRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil, domain.ErrInvalidRefreshToken
		}

		return nil, err
	}

	if !refToken.IsActive(time.Now().UTC()) {
		if refToken.RevokedAt == nil {
			return nil, domain.ErrInvalidRefreshToken
		}

		if err := s.repo.Auth.RevokeUserTokens(ctx, refToken.UserID); err != nil {
			return nil, err
		}

		return nil, domain.ErrInvalidRefreshToken
	}

	user, err := s.repo.Auth.GetUserByID(ctx, refToken.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidRefreshToken
		}

		return nil, err
	}

	pair, err := s.tokens.NewPair(user)
	if err != nil {
		return nil, err
	}

	next := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: token.Hash(pair.Refresh),
		ExpiresAt: pair.RefreshExpiresAt,
	}

	if err := s.repo.Auth.RotateRefreshToken(ctx, hash, next); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil, domain.ErrInvalidRefreshToken
		}

		return nil, err
	}

	return pair, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return domain.ErrInvalidRefreshToken
	}

	if err := s.repo.Auth.RevokeRefreshToken(ctx, token.Hash(refreshToken)); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return domain.ErrInvalidRefreshToken
		}

		return err
	}

	return nil
}

func (s *AuthService) ValidateToken(accessToken string) (*token.Claims, error) {
	if accessToken == "" {
		return nil, domain.ErrInvalidToken
	}

	return s.tokens.ParseAccess(accessToken)
}

func (s *AuthService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.Auth.DeleteUser(ctx, id)
}

func (s *AuthService) createUser(ctx context.Context, email, plainPassword string, role models.Role) (*models.User, error) {
	email, plainPassword, err := normalizeCredentials(email, plainPassword)
	if err != nil {
		return nil, err
	}

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.Auth.CreateUser(ctx, models.User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	})
}

func normalizeCredentials(email, plainPassword string) (string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", "", domain.ErrEmailRequired
	}

	if plainPassword == "" {
		return "", "", domain.ErrPasswordRequired
	}

	return email, plainPassword, nil
}
