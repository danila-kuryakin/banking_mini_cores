package adapters

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/repository"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

type Customer struct {
	customerv1.UnimplementedCustomerServiceServer
	repo *repository.Repository
	// events может быть nil - это штатный режим работы без Kafka,
	// методы продюсера в таком случае ничего не делают.
	events *kafka.Producer
	log    *slog.Logger
}

func NewCustomer(db *pgxpool.Pool, events *kafka.Producer, log *slog.Logger) *Customer {
	return &Customer{
		repo:   repository.NewRepository(db),
		events: events,
		log:    log,
	}
}

func (s *Customer) GetCustomer(ctx context.Context, in *customerv1.GetCustomerRequest) (*customerv1.GetCustomerResponse, error) {
	customerId := in.CustomerId

	return &customerv1.GetCustomerResponse{
		Customer: &customerv1.Customer{
			CustomerId: customerId,
			UserId:     "",
			Status:     customerv1.CustomerStatus_CUSTOMER_STATUS_UNSPECIFIED,
			Profile:    nil,
			CreatedAt:  timestamppb.New(time.Now().UTC()),
			UpdatedAt:  timestamppb.New(time.Now().UTC()),
		},
	}, s.repo.Customer.GetCustomer()
}
func (s *Customer) UpdateProfile(ctx context.Context, in *customerv1.UpdateProfileRequest) (*customerv1.UpdateProfileResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *Customer) GetCustomerStatus(ctx context.Context, in *customerv1.GetCustomerStatusRequest) (*customerv1.GetCustomerStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *Customer) ListCustomers(ctx context.Context, in *customerv1.ListCustomersRequest) (*customerv1.ListCustomersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
