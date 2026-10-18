package server

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

var _ interfaces.LedgerService = (*fakeLedgerService)(nil)

// listCall - аргументы одного вызова ListTransactions.
type listCall struct {
	userID    *uuid.UUID
	accountID uuid.UUID
	page      models.Page
}

// fakeLedgerService - фейк сервисного слоя с заданными ответами и записью вызовов.
type fakeLedgerService struct {
	transferTx  *models.Transaction
	transferErr error
	depositTx   *models.Transaction
	depositErr  error
	getTx       *models.Transaction
	getErr      error
	listTxs     []*models.Transaction
	listErr     error
	balance     *models.Balance
	balanceErr  error

	transferCalls []models.TransferCommand
	depositCalls  []models.DepositCommand
	getCalls      []uuid.UUID
	listCalls     []listCall
	balanceCalls  []uuid.UUID
}

// newFailingLedgerService - фейк, у которого все методы возвращают одну ошибку и nil-результат.
func newFailingLedgerService(err error) *fakeLedgerService {
	return &fakeLedgerService{
		transferErr: err,
		depositErr:  err,
		getErr:      err,
		listErr:     err,
		balanceErr:  err,
	}
}

func (f *fakeLedgerService) Transfer(_ context.Context, cmd models.TransferCommand) (*models.Transaction, error) {
	f.transferCalls = append(f.transferCalls, cmd)
	return f.transferTx, f.transferErr
}

func (f *fakeLedgerService) Deposit(_ context.Context, cmd models.DepositCommand) (*models.Transaction, error) {
	f.depositCalls = append(f.depositCalls, cmd)
	return f.depositTx, f.depositErr
}

func (f *fakeLedgerService) GetTransaction(_ context.Context, transactionID uuid.UUID) (*models.Transaction, error) {
	f.getCalls = append(f.getCalls, transactionID)
	return f.getTx, f.getErr
}

func (f *fakeLedgerService) ListTransactions(_ context.Context, userID *uuid.UUID, accountID uuid.UUID, page models.Page) ([]*models.Transaction, error) {
	var userIDCopy *uuid.UUID
	if userID != nil {
		copied := *userID
		userIDCopy = &copied
	}
	f.listCalls = append(f.listCalls, listCall{userID: userIDCopy, accountID: accountID, page: page})
	return f.listTxs, f.listErr
}

func (f *fakeLedgerService) GetBalance(_ context.Context, accountID uuid.UUID) (*models.Balance, error) {
	f.balanceCalls = append(f.balanceCalls, accountID)
	return f.balance, f.balanceErr
}

// totalCalls - сколько раз фейк вызывали по всем методам.
func (f *fakeLedgerService) totalCalls() int {
	return len(f.transferCalls) + len(f.depositCalls) + len(f.getCalls) + len(f.listCalls) + len(f.balanceCalls)
}

// newTestServer собирает настоящий LedgerServer поверх фейка и логгер с буфером.
func newTestServer(fake *fakeLedgerService) (*LedgerServer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(buf, nil))
	return NewLedgerServer(&service.Service{Ledger: fake}, log), buf
}
