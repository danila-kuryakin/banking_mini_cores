package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/repository"
)

type Service struct {
	Customer *CustomerService
}

func NewService(repo *repository.Repository) *Service {
	return &Service{
		Customer: NewCustomerService(repo),
	}
}
