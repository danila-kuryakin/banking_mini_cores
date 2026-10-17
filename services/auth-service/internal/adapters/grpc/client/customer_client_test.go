package client

import (
	"context"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/mocks"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/customer/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateProfile_Success(t *testing.T) {
	grpcClient := &mocks.CustomerServiceClientMock{}
	defer grpcClient.AssertExpectations(t)
	userID := uuid.New()
	var gotReq *customerv1.CreateProfileRequest
	grpcClient.On("CreateProfile", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { gotReq = args.Get(1).(*customerv1.CreateProfileRequest) }).
		Return(&customerv1.CreateProfileResponse{UserId: userID.String()}, nil).Once()

	err := NewCustomerClient(grpcClient).CreateProfile(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, gotReq)
	assert.Equal(t, userID.String(), gotReq.GetUserId())
}

func TestCreateProfile_AlreadyExists(t *testing.T) {
	grpcClient := &mocks.CustomerServiceClientMock{}
	defer grpcClient.AssertExpectations(t)
	grpcClient.On("CreateProfile", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.AlreadyExists, "exists")).Once()

	err := NewCustomerClient(grpcClient).CreateProfile(context.Background(), uuid.New())

	require.ErrorIs(t, err, domain.ErrProfileAlreadyExists)
}

func TestCreateProfile_Unavailable(t *testing.T) {
	grpcClient := &mocks.CustomerServiceClientMock{}
	defer grpcClient.AssertExpectations(t)
	grpcClient.On("CreateProfile", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.Unavailable, "down")).Once()

	err := NewCustomerClient(grpcClient).CreateProfile(context.Background(), uuid.New())

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrProfileAlreadyExists)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}
