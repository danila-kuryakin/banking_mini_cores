package client

import (
	"context"
	"errors"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/customer/v1"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/ledger/v1"
	"google.golang.org/grpc"
)

// errNotImplemented - ответ методов фейка, которые юнит не использует
var errNotImplemented = errors.New("not implemented")

var _ customerv1.CustomerServiceClient = (*fakeCustomerServiceClient)(nil)

// fakeCustomerServiceClient - фейк customer-service: отдаёт заданный статус или ошибку и пишет user_id запросов
type fakeCustomerServiceClient struct {
	status       customerv1.CustomerStatus
	getStatusErr error
	userIDs      []string
}

func (f *fakeCustomerServiceClient) CreateProfile(_ context.Context, _ *customerv1.CreateProfileRequest, _ ...grpc.CallOption) (*customerv1.CreateProfileResponse, error) {
	return nil, errNotImplemented
}

func (f *fakeCustomerServiceClient) GetCustomer(_ context.Context, _ *customerv1.GetCustomerRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	return nil, errNotImplemented
}

func (f *fakeCustomerServiceClient) GetCustomerStatus(_ context.Context, in *customerv1.GetCustomerStatusRequest, _ ...grpc.CallOption) (*customerv1.GetCustomerStatusResponse, error) {
	f.userIDs = append(f.userIDs, in.GetUserId())
	if f.getStatusErr != nil {
		return nil, f.getStatusErr
	}
	return &customerv1.GetCustomerStatusResponse{Status: f.status}, nil
}

func (f *fakeCustomerServiceClient) UpdateProfile(_ context.Context, _ *customerv1.UpdateProfileRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	return nil, errNotImplemented
}

func (f *fakeCustomerServiceClient) ListCustomers(_ context.Context, _ *customerv1.ListCustomersRequest, _ ...grpc.CallOption) (*customerv1.ListCustomersResponse, error) {
	return nil, errNotImplemented
}

var _ ledgerv1.LedgerServiceClient = (*fakeLedgerServiceClient)(nil)

// fakeLedgerServiceClient - фейк ledger-service: отдаёт заданный остаток или ошибку и пишет account_id запросов
type fakeLedgerServiceClient struct {
	resp          *ledgerv1.GetBalanceResponse
	getBalanceErr error
	accountIDs    []string
}

func (f *fakeLedgerServiceClient) Transfer(_ context.Context, _ *ledgerv1.TransferRequest, _ ...grpc.CallOption) (*ledgerv1.Transaction, error) {
	return nil, errNotImplemented
}

func (f *fakeLedgerServiceClient) Deposit(_ context.Context, _ *ledgerv1.DepositRequest, _ ...grpc.CallOption) (*ledgerv1.Transaction, error) {
	return nil, errNotImplemented
}

func (f *fakeLedgerServiceClient) GetTransaction(_ context.Context, _ *ledgerv1.GetTransactionRequest, _ ...grpc.CallOption) (*ledgerv1.Transaction, error) {
	return nil, errNotImplemented
}

func (f *fakeLedgerServiceClient) ListTransactions(_ context.Context, _ *ledgerv1.ListTransactionsRequest, _ ...grpc.CallOption) (*ledgerv1.ListTransactionsResponse, error) {
	return nil, errNotImplemented
}

func (f *fakeLedgerServiceClient) GetBalance(_ context.Context, in *ledgerv1.GetBalanceRequest, _ ...grpc.CallOption) (*ledgerv1.GetBalanceResponse, error) {
	f.accountIDs = append(f.accountIDs, in.GetAccountId())
	if f.getBalanceErr != nil {
		return nil, f.getBalanceErr
	}
	return f.resp, nil
}
