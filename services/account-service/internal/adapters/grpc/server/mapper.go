package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
)

var statusPairs = []struct {
	domain models.AccountStatus
	proto  accountv1.AccountStatus
}{
	{models.STATUS_ACTIVE, accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE},
	{models.STATUS_BLOCKED, accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED},
	{models.STATUS_CLOSED, accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED},
}

var statusToProtoMap = buildStatusMap()

func buildStatusMap() map[models.AccountStatus]accountv1.AccountStatus {
	toProto := make(map[models.AccountStatus]accountv1.AccountStatus, len(statusPairs))

	for _, pair := range statusPairs {
		toProto[pair.domain] = pair.proto
	}

	return toProto
}

func accountToProto(account *models.Account) *accountv1.Account {
	return &accountv1.Account{
		Id:              account.ID.String(),
		UserId:          account.UserID.String(),
		Number:          account.Number,
		Currency:        string(account.Currency),
		Status:          statusToProtoMap[account.Status],
		StatusChangedAt: timestamppb.New(account.StatusChangedAt),
		CreatedAt:       timestamppb.New(account.CreatedAt),
		UpdatedAt:       timestamppb.New(account.UpdatedAt),
	}
}

func accountsToProto(accounts []*models.Account) []*accountv1.Account {
	out := make([]*accountv1.Account, 0, len(accounts))

	for _, account := range accounts {
		out = append(out, accountToProto(account))
	}

	return out
}

// statusChangesToProto: пустой From в карте не найдётся и уйдёт как ACCOUNT_STATUS_UNSPECIFIED.
func statusChangesToProto(changes []*models.StatusChange) []*accountv1.StatusChange {
	out := make([]*accountv1.StatusChange, 0, len(changes))

	for _, change := range changes {
		item := &accountv1.StatusChange{
			Id:         change.ID,
			AccountId:  change.AccountID.String(),
			FromStatus: statusToProtoMap[change.From],
			ToStatus:   statusToProtoMap[change.To],
			Reason:     change.Reason,
			CreatedAt:  timestamppb.New(change.CreatedAt),
		}

		if change.ActorID != nil {
			item.ActorId = change.ActorID.String()
		}

		out = append(out, item)
	}

	return out
}
