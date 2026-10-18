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
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
)

type AuthService struct {
	repo      *repository.Repository
	tokens    *token.Manager
	customers interfaces.CustomerProfiles
	hasher    interfaces.PasswordHasher
	login     config.Login
}

func NewAuthService(
	repo *repository.Repository,
	tokens *token.Manager,
	customers interfaces.CustomerProfiles,
	hasher interfaces.PasswordHasher,
	login config.Login,
) *AuthService {
	return &AuthService{
		repo:      repo,
		tokens:    tokens,
		customers: customers,
		hasher:    hasher,
		login:     login,
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
		if delErr := s.repo.Auth.DeleteUser(ctx, user.ID); delErr != nil && !errors.Is(delErr, domain.ErrUserNotFound) {
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

	existing, err := s.repo.Auth.GetUserByEmail(ctx, normalizedEmail)
	if err == nil {
		if existing.Role != domain.ROLE_ADMIN {
			return false, domain.ErrAdminEmailNotAdmin
		}

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

	now := time.Now().UTC()

	lockedUntil, err := s.repo.LoginAttempts.LockedUntil(ctx, email, now)
	if err != nil {
		return nil, nil, err
	}

	if lockedUntil != nil {
		return nil, nil, domain.ErrLoginLocked
	}

	user, err := s.repo.Auth.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, nil, s.registerLoginFailure(ctx, email, now)
		}
		return nil, nil, err
	}

	if err := s.hasher.Verify(ctx, user.PasswordHash, plainPassword); err != nil {
		if errors.Is(err, domain.ErrPasswordHasherBusy) {
			return nil, nil, err
		}

		return nil, nil, s.registerLoginFailure(ctx, email, now)
	}

	if err := s.repo.LoginAttempts.Reset(ctx, email); err != nil {
		return nil, nil, err
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

func (s *AuthService) registerLoginFailure(ctx context.Context, email string, now time.Time) error {
	failure := models.LoginFailure{
		At:          now,
		WindowStart: now.Add(-s.login.Window),
		MaxAttempts: s.login.MaxFailedAttempts,
		LockUntil:   now.Add(s.login.LockDuration),
	}

	if err := s.repo.LoginAttempts.RegisterFailure(ctx, email, failure); err != nil {
		return err
	}

	return domain.ErrInvalidCredentials
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
			return nil
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

	hash, err := s.hasher.Hash(ctx, plainPassword)
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
