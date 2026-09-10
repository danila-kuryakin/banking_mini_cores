package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/customer/v1"
)

type CustomerClient struct {
	client customerv1.CustomerServiceClient
}

func NewCustomerClient(client customerv1.CustomerServiceClient) *CustomerClient {
	return &CustomerClient{client: client}
}

func (c *CustomerClient) CreateProfile(ctx context.Context, userID uuid.UUID) error {
	_, err := c.client.CreateProfile(ctx, &customerv1.CreateProfileRequest{
		UserId: userID.String(),
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return domain.ErrProfileAlreadyExists
		}

		return fmt.Errorf("create customer profile: %w", err)
	}

	return nil
}
