package adapters

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	repo *repository.Repository
	log  *slog.Logger
}

func NewAuth(db *pgxpool.Pool, log *slog.Logger) *AuthServer {
	return &AuthServer{
		repo: repository.NewRepository(db),
		log:  log,
	}
}

func (s *AuthServer) Register(ctx context.Context, in *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	email := in.Email

	return &authv1.RegisterResponse{
		User: &authv1.User{
			Email:      email,
			Role:       commonv1.Role_ROLE_UNSPECIFIED,
			CustomerId: "",
			CreatedAt:  timestamppb.New(time.Now()),
		},
	}, s.repo.Auth.Register()
}
func (s *AuthServer) Login(ctx context.Context, in *authv1.LoginRequest) (*authv1.LoginResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AuthServer) Refresh(ctx context.Context, in *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AuthServer) Logout(ctx context.Context, in *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AuthServer) ValidateToken(ctx context.Context, in *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AuthServer) CreateOfficer(ctx context.Context, in *authv1.CreateOfficerRequest) (*authv1.CreateOfficerResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
