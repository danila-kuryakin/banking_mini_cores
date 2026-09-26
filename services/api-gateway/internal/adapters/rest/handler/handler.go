package handler

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
)

type Handler struct {
	Auth       *AuthHandler
	Customer   *CustomerHandler
	Filestore  *FilestoreHandler
	Kyc        *KycHandler
	Account    *AccountHandler
	Ledger     *LedgerHandler
	Middleware *Middleware
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		Auth:       NewAuthHandler(service),
		Customer:   NewCustomerHandler(service),
		Filestore:  NewFilestoreHandler(service),
		Kyc:        NewKycHandler(service),
		Account:    NewAccountHandler(service),
		Ledger:     NewLedgerHandler(service),
		Middleware: NewMiddleware(service),
	}
}
