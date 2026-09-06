package service

import (
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/auth"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/grpc"
)

type Service struct {
	Auth     *AuthService
	Customer *CustomerService
}

func NewService(client *grpc.GRPCClients, verifier *auth.Verifier, timeout time.Duration) *Service {
	return &Service{
		Auth:     NewAuthService(client.Auth, client.Customer, verifier, timeout),
		Customer: NewCustomerService(client.Customer, timeout),
	}
}
