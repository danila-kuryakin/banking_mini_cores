package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

type CustomerServer struct {
	customerv1.UnimplementedCustomerServiceServer
	service *service.Service
	log     *slog.Logger
	timeout time.Duration
}

func NewCustomerServer(service *service.Service, log *slog.Logger, timeout time.Duration) *CustomerServer {
	return &CustomerServer{
		service: service,
		log:     log,
		timeout: timeout,
	}
}

func (s *CustomerServer) CreateProfile(ctx context.Context, in *customerv1.CreateProfileRequest) (*customerv1.CreateProfileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	customer, err := s.service.Customer.CreateProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerExists) {
			return nil, domain.ErrCustomerAlreadyExists
		}

		s.log.Error("failed to create customer profile", "user_id", in.UserId, "error", err)

		return nil, domain.ErrCreateProfileFailed
	}

	return &customerv1.CreateProfileResponse{
		Id:              customer.ID.String(),
		UserId:          customer.UserID.String(),
		Status:          statusToProto(customer.Status),
		StatusChangedAt: timestamppb.New(customer.StatusChangedAt),
		CreatedAt:       timestamppb.New(customer.CreatedAt),
		UpdatedAt:       timestamppb.New(customer.UpdatedAt),
	}, nil
}

func (s *CustomerServer) GetCustomer(ctx context.Context, in *customerv1.GetCustomerRequest) (*customerv1.Customer, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrCustomerIDMustBeUUID
	}

	customer, err := s.service.Customer.GetCustomer(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			return nil, domain.ErrCustomerMissing
		}

		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to get customer", "customer_id", in.UserId, "error", err)

		return nil, status.Error(codes.Internal, "failed to get customer")
	}

	return &customerv1.Customer{
		Id:        customer.ID.String(),
		UserId:    customer.UserID.String(),
		Status:    statusToProto(customer.Status),
		CreatedAt: timestamppb.New(customer.CreatedAt),
		UpdatedAt: timestamppb.New(customer.UpdatedAt),
		Profile: &customerv1.Profile{
			FirstName:   customer.Profile.FirstName,
			LastName:    customer.Profile.LastName,
			BirthDate:   dateToProto(customer.Profile.BirthDate),
			Citizenship: customer.Profile.Citizenship,
			Phone:       customer.Profile.Phone,
		},
	}, nil
}

func (s *CustomerServer) UpdateProfile(ctx context.Context, in *customerv1.UpdateProfileRequest) (*customerv1.Customer, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrCustomerIDMustBeUUID
	}

	var profile models.Profile

	if in.Profile != nil {
		profile = models.Profile{
			FirstName:   in.Profile.FirstName,
			LastName:    in.Profile.LastName,
			BirthDate:   dateFromProto(in.Profile.BirthDate),
			Citizenship: in.Profile.Citizenship,
			Phone:       in.Profile.Phone,
		}
	}

	customer, err := s.service.Customer.UpdateProfile(ctx, userID, profile)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			return nil, domain.ErrCustomerMissing
		}

		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to update profile", "id", in.UserId, "error", err)

		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &customerv1.Customer{
		Id:              customer.ID.String(),
		UserId:          customer.UserID.String(),
		Status:          statusToProto(customer.Status),
		StatusChangedAt: timestamppb.New(customer.StatusChangedAt),
		Profile:         profileToProto(customer.Profile),
		CreatedAt:       timestamppb.New(customer.CreatedAt),
		UpdatedAt:       timestamppb.New(customer.UpdatedAt),
	}, nil
}

func (s *CustomerServer) GetCustomerStatus(ctx context.Context, in *customerv1.GetCustomerStatusRequest) (*customerv1.GetCustomerStatusResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrCustomerIDMustBeUUID
	}

	customerStatus, changedAt, err := s.service.Customer.GetCustomerStatus(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			return nil, domain.ErrCustomerMissing
		}

		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to get customer status", "customer_id", in.UserId, "error", err)

		return nil, status.Error(codes.Internal, "failed to get customer status")
	}

	return &customerv1.GetCustomerStatusResponse{
		Status:          statusToProto(*customerStatus),
		StatusChangedAt: timestamppb.New(*changedAt),
	}, nil
}

func (s *CustomerServer) ListCustomers(ctx context.Context, in *customerv1.ListCustomersRequest) (*customerv1.ListCustomersResponse, error) {

	customers, err := s.service.Customer.ListCustomers(ctx, int(in.Limit), int(in.Offset))
	if err != nil {
		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to list customers", "error", err)

		return nil, domain.ErrListCustomersFailed
	}

	items := make([]*customerv1.Customer, 0, len(customers))
	for _, customer := range customers {
		items = append(items, &customerv1.Customer{
			Id:              customer.ID.String(),
			UserId:          customer.UserID.String(),
			Status:          statusToProto(customer.Status),
			StatusChangedAt: timestamppb.New(customer.StatusChangedAt),
			Profile:         profileToProto(customer.Profile),
			CreatedAt:       timestamppb.New(customer.CreatedAt),
			UpdatedAt:       timestamppb.New(customer.UpdatedAt),
		})
	}

	return &customerv1.ListCustomersResponse{
		Customers: items,
	}, nil
}
