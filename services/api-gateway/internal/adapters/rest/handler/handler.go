package handler

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
)

type Handler struct {
	Auth       *AuthHandler
	Customer   *CustomerHandler
	Middleware *Middleware
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		Auth:       NewAuthHandler(service),
		Customer:   NewCustomerHandler(service),
		Middleware: NewMiddleware(service),
	}
}
