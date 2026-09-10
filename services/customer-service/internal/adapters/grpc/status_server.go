package grpc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

type StatusServer struct {
	customerv1.UnimplementedCustomerStatusServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewStatusServer(service *service.Service, log *slog.Logger) *StatusServer {
	return &StatusServer{
		service: service,
		log:     log,
	}
}

func (s *StatusServer) SetStatus(ctx context.Context, in *customerv1.SetStatusRequest) (*customerv1.SetStatusResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	actorID, err := parseActorID(in.ActorId)
	if err != nil {
		return nil, domain.ErrActorIDMustBeUUID
	}

	change, err := s.service.Customer.SetStatus(ctx, userID, statusFromProto(in.Status), in.Reason, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			return nil, domain.ErrCustomerMissing
		}

		if errors.Is(err, domain.ErrStatusConflict) {
			return nil, domain.ErrStatusChanged
		}

		if errors.Is(err, domain.ErrStatusTransitionForbidden) {
			s.log.Info("status transition rejected", "user_id", in.UserId, "error", err)

			return nil, domain.ErrStatusTransitionForbidden
		}

		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to set customer status", "user_id", in.UserId, "status", in.Status.String(), "error", err)

		return nil, domain.ErrSetStatusFailed
	}

	return &customerv1.SetStatusResponse{
		Status:          statusToProto(change.Current),
		PreviousStatus:  statusToProto(change.Previous),
		StatusChangedAt: timestamppb.New(change.ChangedAt),
	}, nil
}

// parseActorID разбирает необязательный идентификатор инициатора. Пустая
// строка - системный переход, а не ошибка.
func parseActorID(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}
