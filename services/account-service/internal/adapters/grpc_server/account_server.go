package grpc_server

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/repository"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
)

type AccountServer struct {
	accountv1.UnimplementedAccountServiceServer
	repo *repository.Repository
	log  *slog.Logger
}

func NewAccount(db *pgxpool.Pool, log *slog.Logger) *AccountServer {
	return &AccountServer{
		repo: repository.NewRepository(db),
		log:  log,
	}
}

func (s *AccountServer) OpenAccount(ctx context.Context, in *accountv1.OpenAccountRequest) (*accountv1.OpenAccountResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AccountServer) GetAccount(ctx context.Context, in *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	accountId := in.AccountId

	return &accountv1.GetAccountResponse{
		Account: &accountv1.Account{
			AccountId:     accountId,
			CustomerId:    "",
			AccountNumber: "",
			Currency:      "",
			Status:        accountv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED,
			BlockReason:   "",
			OpenedAt:      timestamppb.New(time.Now().UTC()),
			ClosedAt:      nil,
		},
	}, s.repo.Account.GetAccount()
}
func (s *AccountServer) ListAccounts(ctx context.Context, in *accountv1.ListAccountsRequest) (*accountv1.ListAccountsResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AccountServer) BlockAccount(ctx context.Context, in *accountv1.BlockAccountRequest) (*accountv1.BlockAccountResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AccountServer) CloseAccount(ctx context.Context, in *accountv1.CloseAccountRequest) (*accountv1.CloseAccountResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AccountServer) GetBalance(ctx context.Context, in *accountv1.GetBalanceRequest) (*accountv1.GetBalanceResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
