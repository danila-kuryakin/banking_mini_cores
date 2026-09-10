package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
)

type Service struct {
	Auth *AuthService
}

func NewService(
	repo *repository.Repository,
	tokens *token.Manager,
	customers interfaces.CustomerProfiles,
) *Service {
	return &Service{
		Auth: NewAuthService(repo, tokens, customers),
	}
}
