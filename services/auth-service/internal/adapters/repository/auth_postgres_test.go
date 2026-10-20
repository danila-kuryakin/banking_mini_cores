package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
)

func TestDeleteUser_DeletesExisting(t *testing.T) {
	repo := NewAuthRepo(testPool(t))
	ctx := context.Background()
	created, err := repo.CreateUser(ctx, models.User{
		Email:        uuid.NewString() + TEST_EMAIL_DOMAIN,
		PasswordHash: "hash",
		Role:         domain.ROLE_CLIENT,
	})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteUser(ctx, created.ID))

	_, err = repo.GetUserByID(ctx, created.ID)
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestDeleteUser_NotFound(t *testing.T) {
	repo := NewAuthRepo(testPool(t))

	err := repo.DeleteUser(context.Background(), uuid.New())

	require.ErrorIs(t, err, domain.ErrUserNotFound)
}
