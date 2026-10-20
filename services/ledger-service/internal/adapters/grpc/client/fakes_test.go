package client

import (
	"context"

	"google.golang.org/grpc"

	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/account/v1"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/antifraud/v1"
)

// fakeAccountServiceClient реализует только GetAccount и GetAccountByNumber, остальные методы не нужны
type fakeAccountServiceClient struct {
	accountv1.AccountServiceClient
	requests       []*accountv1.GetAccountRequest
	numberRequests []*accountv1.GetAccountByNumberRequest
	resp           *accountv1.Account
	err            error
}

var _ accountv1.AccountServiceClient = (*fakeAccountServiceClient)(nil)

func (f *fakeAccountServiceClient) GetAccount(_ context.Context, in *accountv1.GetAccountRequest, _ ...grpc.CallOption) (*accountv1.Account, error) {
	f.requests = append(f.requests, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func (f *fakeAccountServiceClient) GetAccountByNumber(_ context.Context, in *accountv1.GetAccountByNumberRequest, _ ...grpc.CallOption) (*accountv1.Account, error) {
	f.numberRequests = append(f.numberRequests, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

// fakeAntifraudServiceClient реализует только CheckTransfer
type fakeAntifraudServiceClient struct {
	antifraudv1.AntifraudServiceClient
	requests []*antifraudv1.CheckTransferRequest
	resp     *antifraudv1.CheckTransferResponse
	err      error
}

var _ antifraudv1.AntifraudServiceClient = (*fakeAntifraudServiceClient)(nil)

func (f *fakeAntifraudServiceClient) CheckTransfer(_ context.Context, in *antifraudv1.CheckTransferRequest, _ ...grpc.CallOption) (*antifraudv1.CheckTransferResponse, error) {
	f.requests = append(f.requests, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}
