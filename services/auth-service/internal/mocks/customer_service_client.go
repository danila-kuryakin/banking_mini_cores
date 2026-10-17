package mocks

import (
	"context"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/customer/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var _ customerv1.CustomerServiceClient = (*CustomerServiceClientMock)(nil)

// CustomerServiceClientMock - мок gRPC-клиента customer-service, через mock реализован только CreateProfile.
type CustomerServiceClientMock struct {
	mock.Mock
	customerv1.CustomerServiceClient
}

// CreateProfile передаёт opts в mock одним срезом.
func (m *CustomerServiceClientMock) CreateProfile(ctx context.Context, in *customerv1.CreateProfileRequest, opts ...grpc.CallOption) (*customerv1.CreateProfileResponse, error) {
	args := m.Called(ctx, in, opts)
	if v := args.Get(0); v != nil {
		return v.(*customerv1.CreateProfileResponse), args.Error(1)
	}
	return nil, args.Error(1)
}
