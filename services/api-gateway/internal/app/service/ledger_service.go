package service

import (
	"context"
	"time"

	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/ledger/v1"
)

type LedgerService struct {
	ledgerClient ledgerv1.LedgerServiceClient
	timeout      time.Duration
}

func NewLedgerService(ledgerCli ledgerv1.LedgerServiceClient, timeout time.Duration) *LedgerService {
	return &LedgerService{
		ledgerClient: ledgerCli,
		timeout:      timeout,
	}
}

func (s LedgerService) Transfer(ctx context.Context, in *ledgerv1.TransferRequest) (*ledgerv1.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.ledgerClient.Transfer(ctx, in)
}

func (s LedgerService) ListTransactions(ctx context.Context, in *ledgerv1.ListTransactionsRequest) ([]*ledgerv1.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.ledgerClient.ListTransactions(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Transactions, nil
}

func (s LedgerService) Deposit(ctx context.Context, in *ledgerv1.DepositRequest) (*ledgerv1.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.ledgerClient.Deposit(ctx, in)
}

func (s LedgerService) GetTransaction(ctx context.Context, transactionID string) (*ledgerv1.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.ledgerClient.GetTransaction(ctx, &ledgerv1.GetTransactionRequest{TransactionId: transactionID})
}
