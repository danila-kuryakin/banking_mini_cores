package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	service *service.Service
	log     *slog.Logger
	timeout time.Duration
}

func NewAuthServer(service *service.Service, log *slog.Logger, timeout time.Duration) *AuthServer {
	return &AuthServer{
		service: service,
		log:     log,
		timeout: timeout,
	}
}

func (s *AuthServer) Register(ctx context.Context, in *authv1.RegisterRequest) (*authv1.User, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	user, err := s.service.Auth.Register(ctx, in.Email, in.Password)
	if err != nil {
		return nil, s.credentialsError(err, "failed to register user")
	}
	return &authv1.User{
		Id:        user.ID.String(),
		Email:     user.Email,
		Role:      roleToProto(user.Role),
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, in *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	user, pair, err := s.service.Auth.Login(ctx, in.Email, in.Password)
	if err != nil {
		return nil, s.credentialsError(err, "failed to login")
	}

	return &authv1.LoginResponse{
		User: &authv1.User{
			Id:        user.ID.String(),
			Email:     user.Email,
			Role:      roleToProto(user.Role),
			CreatedAt: timestamppb.New(user.CreatedAt),
		},
		Tokens: &authv1.TokenPair{
			AccessToken:      pair.Access,
			RefreshToken:     pair.Refresh,
			AccessExpiresAt:  timestamppb.New(pair.AccessExpiresAt),
			RefreshExpiresAt: timestamppb.New(pair.RefreshExpiresAt),
		},
	}, nil
}

func (s *AuthServer) Refresh(ctx context.Context, in *authv1.RefreshRequest) (*authv1.TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	pair, err := s.service.Auth.Refresh(ctx, in.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		s.log.Error("failed to refresh tokens", "error", err)

		return nil, domain.ErrRefreshFailed
	}

	return &authv1.TokenPair{
		AccessToken:      pair.Access,
		RefreshToken:     pair.Refresh,
		AccessExpiresAt:  timestamppb.New(pair.AccessExpiresAt),
		RefreshExpiresAt: timestamppb.New(pair.RefreshExpiresAt),
	}, nil
}

func (s *AuthServer) Logout(ctx context.Context, in *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.service.Auth.Logout(ctx, in.RefreshToken); err != nil {
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		s.log.Error("failed to logout", "error", err)

		return nil, domain.ErrLogoutFailed
	}

	return &authv1.LogoutResponse{}, nil
}

func (s *AuthServer) ValidateToken(_ context.Context, in *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims, err := s.service.Auth.ValidateToken(in.AccessToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return &authv1.ValidateTokenResponse{Valid: false}, nil
		}

		s.log.Error("failed to validate token", "error", err)

		return nil, domain.ErrValidateFailed
	}

	return &authv1.ValidateTokenResponse{
		Valid:     true,
		Id:        claims.UserID.String(),
		Role:      roleToProto(claims.Role),
		ExpiresAt: timestamppb.New(claims.ExpiresAt),
	}, nil
}

func (s *AuthServer) CreateOfficer(ctx context.Context, in *authv1.RegisterRequest) (*authv1.User, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	user, err := s.service.Auth.CreateOfficer(ctx, in.Email, in.Password)
	if err != nil {
		return nil, s.credentialsError(err, "failed to create officer")
	}

	role := roleToProto(user.Role)

	if role != commonv1.Role_ROLE_OFFICER {
		s.log.Error("failed to create officer", "role", role)

		return nil, s.credentialsError(errors.New("invalid role"), "invalid role")
	}

	return &authv1.User{
		Id:        user.ID.String(),
		Email:     user.Email,
		Role:      role,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *AuthServer) DeleteUser(ctx context.Context, in *authv1.DeleteUserRequest) (*authv1.DeleteUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	id, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, domain.ErrIDMustBeUUID
	}

	if err := s.service.Auth.DeleteUser(ctx, id); err != nil {

		s.log.Error("failed to delete user", "id", in.Id, "error", err)

		return nil, domain.ErrDeleteUserFailed
	}

	return &authv1.DeleteUserResponse{}, nil
}
