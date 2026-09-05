package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
)

type Service struct {
	Auth *AuthService
}

func NewService(repo *repository.Repository, tokens *token.Manager) *Service {
	return &Service{
		Auth: NewAuthService(repo, tokens),
	}
}
