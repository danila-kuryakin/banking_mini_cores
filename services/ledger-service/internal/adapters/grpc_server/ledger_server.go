package grpc_server

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/repository"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

type LedgerServer struct {
	ledgerv1.UnimplementedLedgerServiceServer
	repo *repository.Repository
	// events может быть nil - это штатный режим работы без Kafka,
	// методы продюсера в таком случае ничего не делают.
	events *kafka.Producer
	log    *slog.Logger
}

func NewLedger(db *pgxpool.Pool, events *kafka.Producer, log *slog.Logger) *LedgerServer {
	return &LedgerServer{
		repo:   repository.NewRepository(db),
		events: events,
		log:    log,
	}
}

func (s *LedgerServer) Transfer(ctx context.Context, in *ledgerv1.TransferRequest) (*ledgerv1.TransferResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *LedgerServer) Deposit(ctx context.Context, in *ledgerv1.DepositRequest) (*ledgerv1.DepositResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *LedgerServer) GetTransaction(ctx context.Context, in *ledgerv1.GetTransactionRequest) (*ledgerv1.GetTransactionResponse, error) {
	transactionId := in.TransactionId

	return &ledgerv1.GetTransactionResponse{
		Transaction: &ledgerv1.Transaction{
			TransactionId: transactionId,
			Type:          ledgerv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED,
			Status:        ledgerv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED,
			Amount:        nil,
			FromAccountId: "",
			ToAccountId:   "",
			CreatedAt:     timestamppb.New(time.Now().UTC()),
		},
	}, s.repo.Ledger.GetTransaction()
}
func (s *LedgerServer) ListTransactions(ctx context.Context, in *ledgerv1.ListTransactionsRequest) (*ledgerv1.ListTransactionsResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *LedgerServer) GetBalance(ctx context.Context, in *ledgerv1.GetBalanceRequest) (*ledgerv1.GetBalanceResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
