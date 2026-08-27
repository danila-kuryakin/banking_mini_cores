package grpc_server

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/postgres"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	repo   *postgres.Repository
	events *kafka.Producer
	log    *slog.Logger
}

func NewAuth(db *pgxpool.Pool, events *kafka.Producer, log *slog.Logger) *AuthServer {
	return &AuthServer{
		repo:   postgres.NewRepository(db),
		events: events,
		log:    log,
	}
}

func (s *AuthServer) Register(ctx context.Context, in *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if err := s.repo.Auth.Register(); err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC()

	user := &authv1.User{
		Email:      in.Email,
		Role:       commonv1.Role_ROLE_UNSPECIFIED,
		CustomerId: "",
		CreatedAt:  timestamppb.New(createdAt),
	}

	if err := s.events.PublishUserRegistered(ctx, kafka.UserRegistered{
		UserID:     user.UserId,
		Email:      user.Email,
		Role:       user.Role.String(),
		CustomerID: user.CustomerId,
		CreatedAt:  createdAt,
	}); err != nil {
		s.log.Error("kafka: событие о регистрации не опубликовано",
			slog.String("email", user.Email),
			slog.Any("error", err),
		)
	}

	return &authv1.RegisterResponse{User: user}, nil
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
