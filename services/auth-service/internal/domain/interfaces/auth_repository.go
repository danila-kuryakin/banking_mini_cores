package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user models.User) (*models.User, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	AddRefreshToken(ctx context.Context, token models.RefreshToken) error
	GetRefreshToken(ctx context.Context, hash []byte) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, hash []byte) error
	RevokeUserTokens(ctx context.Context, userID uuid.UUID) error
	RotateRefreshToken(ctx context.Context, oldHash []byte, next models.RefreshToken) error
}
