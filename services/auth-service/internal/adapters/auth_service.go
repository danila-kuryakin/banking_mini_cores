package adapters

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
)

// prefix добавляется к каждому сообщению, которое возвращает сервис.
const prefix = "Ответил. "

// Auth реализует echov1.EchoServiceServer.
type Auth struct {
	authv1.UnimplementedAuthServiceServer

	log *slog.Logger
}

// NewEcho создаёт реализацию сервиса.
func NewAuth(log *slog.Logger) *Auth {
	return &Auth{log: log}
}

func (s *Auth) Register(ctx context.Context, in *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {

	email := in.Email

	return &authv1.RegisterResponse{
		User: &authv1.User{
			Email:      email,
			Role:       commonv1.Role_ROLE_UNSPECIFIED,
			CustomerId: "",
			CreatedAt:  timestamppb.New(time.Now()),
		},
	}, nil
}
func (s *Auth) Login(ctx context.Context, in *authv1.LoginRequest) (*authv1.LoginResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *Auth) Refresh(ctx context.Context, in *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *Auth) Logout(ctx context.Context, in *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *Auth) ValidateToken(ctx context.Context, in *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *Auth) CreateOfficer(ctx context.Context, in *authv1.CreateOfficerRequest) (*authv1.CreateOfficerResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
