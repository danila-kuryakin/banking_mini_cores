package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/customer/v1"
)

type CustomerClient struct {
	customers customerv1.CustomerServiceClient
}

func NewCustomerClient(customers customerv1.CustomerServiceClient) *CustomerClient {
	return &CustomerClient{customers: customers}
}

func (c *CustomerClient) IsActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	resp, err := c.customers.GetCustomerStatus(ctx, &customerv1.GetCustomerStatusRequest{UserId: userID.String()})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, domain.ErrCustomerNotFound
		}

		return false, fmt.Errorf("get customer status: %w", err)
	}

	return resp.Status == customerv1.CustomerStatus_ACTIVE, nil
}
