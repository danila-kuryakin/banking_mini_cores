package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/password"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
)

type Publisher interface {
	PublishUserRegistered(ctx context.Context, event models.UserRegistered) error
}

type AuthService struct {
	repo   *repository.Repository
	tokens *token.Manager
}

func NewAuthService(repo *repository.Repository, tokens *token.Manager) *AuthService {
	return &AuthService{
		repo:   repo,
		tokens: tokens,
	}
}

func (s *AuthService) Register(ctx context.Context, email, plainPassword string) (*models.User, error) {
	return s.createUser(ctx, email, plainPassword, domain.ROLE_CLIENT)
}

func (s *AuthService) CreateOfficer(ctx context.Context, email, plainPassword string) (*models.User, error) {
	return s.createUser(ctx, email, plainPassword, domain.ROLE_OFFICER)
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

	return s.repo.Auth.RevokeRefreshToken(ctx, token.Hash(refreshToken))
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
