package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/customer/v1"
)

type CustomerClient struct {
	customers customerv1.CustomerServiceClient
	statuses  customerv1.CustomerStatusServiceClient
}

func NewCustomerClient(
	customers customerv1.CustomerServiceClient,
	statuses customerv1.CustomerStatusServiceClient,
) *CustomerClient {
	return &CustomerClient{customers: customers, statuses: statuses}
}

func (c *CustomerClient) GetProfile(ctx context.Context, userID uuid.UUID) (*models.CustomerProfile, error) {
	resp, err := c.customers.GetCustomer(ctx, &customerv1.GetCustomerRequest{UserId: userID.String()})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, domain.ErrCustomerNotFound
		}

		return nil, fmt.Errorf("get customer: %w", err)
	}

	profile := &models.CustomerProfile{
		Status: customerStatusFromProto(resp.Status),
	}

	if resp.Profile != nil {
		profile.FirstName = resp.Profile.FirstName
		profile.LastName = resp.Profile.LastName
		profile.BirthDate = dateFromProto(resp.Profile.BirthDate)
	}

	return profile, nil
}

func (c *CustomerClient) SetStatus(
	ctx context.Context,
	userID uuid.UUID,
	to models.CustomerStatus,
	reason string,
	actorID *uuid.UUID,
) error {
	req := &customerv1.SetStatusRequest{
		UserId: userID.String(),
		Status: customerStatusToProto(to),
		Reason: reason,
	}

	if actorID != nil {
		req.ActorId = actorID.String()
	}

	if _, err := c.statuses.SetStatus(ctx, req); err != nil {
		if status.Code(err) == codes.NotFound {
			return domain.ErrCustomerNotFound
		}

		return fmt.Errorf("set customer status: %w", err)
	}

	return nil
}
