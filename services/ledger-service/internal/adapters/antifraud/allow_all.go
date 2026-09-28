package antifraud

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

// AllowAll - заглушка до появления antifraud-service: разрешает любой перевод.
type AllowAll struct{}

func NewAllowAll() *AllowAll {
	return &AllowAll{}
}

func (AllowAll) CheckTransfer(context.Context, models.TransferCommand) (models.Verdict, error) {
	return models.Verdict{Allowed: true}, nil
}
