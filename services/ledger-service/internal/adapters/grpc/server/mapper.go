package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

var typeToProto = map[models.TransactionType]ledgerv1.TransactionType{
	models.TYPE_TRANSFER: ledgerv1.TransactionType_TRANSACTION_TYPE_TRANSFER,
	models.TYPE_DEPOSIT:  ledgerv1.TransactionType_TRANSACTION_TYPE_DEPOSIT,
}

var statusToProto = map[models.TransactionStatus]ledgerv1.TransactionStatus{
	models.STATUS_COMPLETED: ledgerv1.TransactionStatus_TRANSACTION_STATUS_COMPLETED,
	models.STATUS_DECLINED:  ledgerv1.TransactionStatus_TRANSACTION_STATUS_DECLINED,
}

var directionToProto = map[models.Direction]ledgerv1.EntryDirection{
	models.DIRECTION_DEBIT:  ledgerv1.EntryDirection_ENTRY_DIRECTION_DEBIT,
	models.DIRECTION_CREDIT: ledgerv1.EntryDirection_ENTRY_DIRECTION_CREDIT,
}

func transactionToProto(tx *models.Transaction) *ledgerv1.Transaction {
	out := &ledgerv1.Transaction{
		Id:            tx.ID.String(),
		Type:          typeToProto[tx.Type],
		Status:        statusToProto[tx.Status],
		FromAccountId: tx.FromAccountID.String(),
		ToAccountId:   tx.ToAccountID.String(),
		Amount:        tx.Amount,
		Currency:      string(tx.Currency),
		DeclineReason: tx.DeclineReason,
		CreatedAt:     timestamppb.New(tx.CreatedAt),
		Entries:       make([]*ledgerv1.Entry, 0, len(tx.Entries)),
	}

	if tx.InitiatorID != nil {
		out.InitiatorId = tx.InitiatorID.String()
	}

	for _, entry := range tx.Entries {
		out.Entries = append(out.Entries, &ledgerv1.Entry{
			Id:        entry.ID.String(),
			AccountId: entry.AccountID.String(),
			Direction: directionToProto[entry.Direction],
			Amount:    entry.Amount,
			Currency:  string(entry.Currency),
		})
	}

	return out
}

func transactionsToProto(transactions []*models.Transaction) []*ledgerv1.Transaction {
	out := make([]*ledgerv1.Transaction, 0, len(transactions))

	for _, tx := range transactions {
		out = append(out, transactionToProto(tx))
	}

	return out
}
