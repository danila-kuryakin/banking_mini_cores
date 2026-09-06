package service

import (
	"context"
	"time"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
)

type CustomerService struct {
	customerClient customerv1.CustomerServiceClient
	timeout        time.Duration
}

func NewCustomerService(customerCli customerv1.CustomerServiceClient, timeout time.Duration) *CustomerService {
	return &CustomerService{
		customerClient: customerCli,
		timeout:        timeout,
	}
}

func (s CustomerService) GetCustomer(ctx context.Context, userID string) (*customerv1.Customer, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.customerClient.GetCustomer(ctx, &customerv1.GetCustomerRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s CustomerService) UpdateProfile(ctx context.Context, userID string, profile *customerv1.Profile) (*customerv1.Customer, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.customerClient.UpdateProfile(ctx, &customerv1.UpdateProfileRequest{
		UserId:  userID,
		Profile: profile,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s CustomerService) GetCustomerStatus(ctx context.Context, userID string) (*customerv1.GetCustomerStatusResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.customerClient.GetCustomerStatus(ctx, &customerv1.GetCustomerStatusRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s CustomerService) ListCustomers(ctx context.Context, in *customerv1.ListCustomersRequest) ([]*customerv1.Customer, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.customerClient.ListCustomers(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Customers, nil
}
