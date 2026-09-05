package service

import (
	"context"
	"errors"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/auth"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/auth/v1"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
)

type AuthService struct {
	authClient     authv1.AuthServiceClient
	customerClient customerv1.CustomerServiceClient
	verifier       *auth.Verifier
	timeout        time.Duration
}

func NewAuthService(
	authCli authv1.AuthServiceClient,
	customerCli customerv1.CustomerServiceClient,
	verifier *auth.Verifier,
	timeout time.Duration,
) *AuthService {
	return &AuthService{
		authClient:     authCli,
		customerClient: customerCli,
		verifier:       verifier,
		timeout:        timeout,
	}
}

func (s AuthService) Register(ctx context.Context, in *authv1.RegisterRequest) (*authv1.User, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	user, err := s.authClient.Register(ctx, in)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerClient.CreateProfile(ctx, &customerv1.CreateProfileRequest{
		UserId: user.Id,
	})
	if err != nil {
		if _, deleteErr := s.authClient.DeleteUser(ctx, &authv1.DeleteUserRequest{Id: user.Id}); deleteErr != nil {
			return nil, errors.Join(err, deleteErr)
		}
		return nil, err
	}

	if customer.UserId != user.Id {
		return nil, errors.New("customer user id does not match")
	}

	return user, nil
}

func (s AuthService) Login(ctx context.Context, in *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	logged, err := s.authClient.Login(ctx, in)
	if err != nil {
		return nil, err
	}

	return logged, nil
}

func (s AuthService) RefreshToken(ctx context.Context, refreshToken string) (*authv1.TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	refreshed, err := s.authClient.Refresh(ctx, &authv1.RefreshRequest{RefreshToken: refreshToken})
	if err != nil {
		return nil, err
	}

	return refreshed, nil
}

func (s AuthService) Logout(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.authClient.Logout(ctx, &authv1.LogoutRequest{RefreshToken: refreshToken})

	return err
}

func (s AuthService) ValidateToken(ctx context.Context, accessToken string) (domain.ValidateTokenOutput, error) {
	claims, err := s.verifier.Verify(ctx, accessToken)
	if err != nil {
		if errors.Is(err, domain.ErrAccessTokenNotValid) {
			return domain.ValidateTokenOutput{Valid: false}, nil
		}

		return domain.ValidateTokenOutput{}, err
	}

	return domain.ValidateTokenOutput{
		Valid:     true,
		ID:        claims.UserID,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func (s AuthService) CreateOfficers(ctx context.Context, in *authv1.RegisterRequest) (*authv1.User, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	created, err := s.authClient.CreateOfficer(ctx, in)
	if err != nil {
		return nil, err
	}

	return created, nil
}
