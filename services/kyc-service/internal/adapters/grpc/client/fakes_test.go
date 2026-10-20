package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/customer/v1"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/filestore/v1"
)

// errFakeUnimplemented - ответ заглушек, которые адаптер вызывать не должен
var errFakeUnimplemented = status.Error(codes.Unimplemented, "fake: method not implemented")

// fakeCustomerServiceClient - фейк CustomerServiceClient: записывает запросы GetCustomer и отдаёт заданный ответ
type fakeCustomerServiceClient struct {
	getCustomerResp  *customerv1.Customer
	getCustomerErr   error
	getCustomerCalls []*customerv1.GetCustomerRequest
}

var _ customerv1.CustomerServiceClient = (*fakeCustomerServiceClient)(nil)

func (f *fakeCustomerServiceClient) CreateProfile(_ context.Context, _ *customerv1.CreateProfileRequest, _ ...grpc.CallOption) (*customerv1.CreateProfileResponse, error) {
	return nil, errFakeUnimplemented
}

func (f *fakeCustomerServiceClient) GetCustomer(_ context.Context, in *customerv1.GetCustomerRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	f.getCustomerCalls = append(f.getCustomerCalls, in)
	if f.getCustomerErr != nil {
		return nil, f.getCustomerErr
	}
	return f.getCustomerResp, nil
}

func (f *fakeCustomerServiceClient) GetCustomerStatus(_ context.Context, _ *customerv1.GetCustomerStatusRequest, _ ...grpc.CallOption) (*customerv1.GetCustomerStatusResponse, error) {
	return nil, errFakeUnimplemented
}

func (f *fakeCustomerServiceClient) UpdateProfile(_ context.Context, _ *customerv1.UpdateProfileRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	return nil, errFakeUnimplemented
}

func (f *fakeCustomerServiceClient) ListCustomers(_ context.Context, _ *customerv1.ListCustomersRequest, _ ...grpc.CallOption) (*customerv1.ListCustomersResponse, error) {
	return nil, errFakeUnimplemented
}

// fakeCustomerStatusServiceClient - фейк CustomerStatusServiceClient: записывает запросы SetStatus
type fakeCustomerStatusServiceClient struct {
	setStatusErr   error
	setStatusCalls []*customerv1.SetStatusRequest
}

var _ customerv1.CustomerStatusServiceClient = (*fakeCustomerStatusServiceClient)(nil)

func (f *fakeCustomerStatusServiceClient) SetStatus(_ context.Context, in *customerv1.SetStatusRequest, _ ...grpc.CallOption) (*customerv1.SetStatusResponse, error) {
	f.setStatusCalls = append(f.setStatusCalls, in)
	if f.setStatusErr != nil {
		return nil, f.setStatusErr
	}
	return &customerv1.SetStatusResponse{}, nil
}

// fakeFileServiceClient - фейк FileServiceClient: записывает запросы HasRequiredFiles и ListFiles
type fakeFileServiceClient struct {
	hasRequiredResp  *filestorev1.HasRequiredFilesResponse
	hasRequiredErr   error
	hasRequiredCalls []*filestorev1.HasRequiredFilesRequest

	listFilesResp  *filestorev1.ListFilesResponse
	listFilesErr   error
	listFilesCalls []*filestorev1.ListFilesRequest
}

var _ filestorev1.FileServiceClient = (*fakeFileServiceClient)(nil)

func (f *fakeFileServiceClient) InitUpload(_ context.Context, _ *filestorev1.InitUploadRequest, _ ...grpc.CallOption) (*filestorev1.InitUploadResponse, error) {
	return nil, errFakeUnimplemented
}

func (f *fakeFileServiceClient) ConfirmUpload(_ context.Context, _ *filestorev1.ConfirmUploadRequest, _ ...grpc.CallOption) (*filestorev1.FileMetadata, error) {
	return nil, errFakeUnimplemented
}

func (f *fakeFileServiceClient) GetDownloadUrl(_ context.Context, _ *filestorev1.GetDownloadUrlRequest, _ ...grpc.CallOption) (*filestorev1.GetDownloadUrlResponse, error) {
	return nil, errFakeUnimplemented
}

func (f *fakeFileServiceClient) ListFiles(_ context.Context, in *filestorev1.ListFilesRequest, _ ...grpc.CallOption) (*filestorev1.ListFilesResponse, error) {
	f.listFilesCalls = append(f.listFilesCalls, in)
	if f.listFilesErr != nil {
		return nil, f.listFilesErr
	}
	return f.listFilesResp, nil
}

func (f *fakeFileServiceClient) HasRequiredFiles(_ context.Context, in *filestorev1.HasRequiredFilesRequest, _ ...grpc.CallOption) (*filestorev1.HasRequiredFilesResponse, error) {
	f.hasRequiredCalls = append(f.hasRequiredCalls, in)
	if f.hasRequiredErr != nil {
		return nil, f.hasRequiredErr
	}
	return f.hasRequiredResp, nil
}

// assertProtoCalls сравнивает записанные фейком запросы с ожидаемыми через proto.Equal
func assertProtoCalls[T proto.Message](t *testing.T, want, got []T) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		assert.Truef(t, proto.Equal(want[i], got[i]), "call %d: want %v, got %v", i, want[i], got[i])
	}
}
