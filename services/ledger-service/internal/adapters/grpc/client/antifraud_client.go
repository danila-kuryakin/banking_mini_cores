package client

import (
	"context"
	"fmt"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/antifraud/v1"
)

type AntifraudClient struct {
	antifraud antifraudv1.AntifraudServiceClient
}

func NewAntifraudClient(antifraud antifraudv1.AntifraudServiceClient) *AntifraudClient {
	return &AntifraudClient{antifraud: antifraud}
}

// CheckTransfer: разрешает только явный ALLOW, неизвестный вердикт считается ошибкой, а не разрешением.
func (c *AntifraudClient) CheckTransfer(ctx context.Context, cmd models.TransferCommand) (models.Verdict, error) {
	resp, err := c.antifraud.CheckTransfer(ctx, &antifraudv1.CheckTransferRequest{
		InitiatorId:   cmd.InitiatorID.String(),
		FromAccountId: cmd.From.String(),
		ToAccountId:   cmd.To.String(),
		Amount:        cmd.Amount,
		Currency:      cmd.Currency,
	})
	if err != nil {
		return models.Verdict{}, fmt.Errorf("check transfer: %w", err)
	}

	switch resp.Verdict {
	case antifraudv1.Verdict_VERDICT_ALLOW:
		return models.Verdict{Allowed: true, CheckID: resp.CheckId}, nil
	case antifraudv1.Verdict_VERDICT_DENY:
		return models.Verdict{Allowed: false, CheckID: resp.CheckId, Reasons: resp.Reasons}, nil
	default:
		return models.Verdict{}, fmt.Errorf("check transfer: unexpected verdict %s", resp.Verdict)
	}
}
