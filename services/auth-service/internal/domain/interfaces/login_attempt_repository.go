package interfaces

import (
	"context"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
)

type LoginAttemptRepository interface {
	LockedUntil(ctx context.Context, email string, now time.Time) (*time.Time, error)
	RegisterFailure(ctx context.Context, email string, failure models.LoginFailure) error
	Reset(ctx context.Context, email string) error
}
