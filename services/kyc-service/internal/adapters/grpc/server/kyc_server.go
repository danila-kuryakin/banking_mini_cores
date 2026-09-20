package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

type KycServer struct {
	kycv1.UnimplementedKycServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewKycServer(service *service.Service, log *slog.Logger) *KycServer {
	return &KycServer{service: service, log: log}
}

func (s *KycServer) SubmitApplication(ctx context.Context, in *kycv1.SubmitApplicationRequest) (*kycv1.Application, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	app, err := s.service.Application.Submit(ctx, userID)
	if err != nil {
		return nil, s.fail("failed to submit application", err, domain.ErrSubmitFailed, "user_id", in.UserId)
	}

	return applicationToProto(app), nil
}

func (s *KycServer) GetApplicationByID(ctx context.Context, in *kycv1.GetApplicationByIDRequest) (*kycv1.Application, error) {
	id, err := uuid.Parse(in.ApplicationId)
	if err != nil {
		return nil, domain.ErrApplicationIDMustBeUUID
	}

	app, err := s.service.Application.GetByID(ctx, id)
	if err != nil {
		return nil, s.fail("failed to get application", err, domain.ErrGetFailed, "application_id", in.ApplicationId)
	}

	return applicationToProto(app), nil
}

func (s *KycServer) GetApplicationByUserID(ctx context.Context, in *kycv1.GetApplicationByUserIDRequest) (*kycv1.Application, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	app, err := s.service.Application.GetLatestByUserID(ctx, userID)
	if err != nil {
		return nil, s.fail("failed to get application", err, domain.ErrGetFailed, "user_id", in.UserId)
	}

	return applicationToProto(app), nil
}

func (s *KycServer) ListApplications(ctx context.Context, in *kycv1.ListApplicationsRequest) (*kycv1.ListApplicationsResponse, error) {
	filter := models.ApplicationFilter{
		Statuses:   statusesFromProto(in.Statuses),
		Unassigned: in.Unassigned,
		Limit:      int(in.Limit),
		Offset:     int(in.Offset),
	}

	if in.ReviewerId != "" {
		reviewerID, err := uuid.Parse(in.ReviewerId)
		if err != nil {
			return nil, domain.ErrOfficerIDMustBeUUID
		}

		filter.ReviewerID = &reviewerID
	}

	apps, err := s.service.Application.List(ctx, filter)
	if err != nil {
		return nil, s.fail("failed to list applications", err, domain.ErrListFailed)
	}

	items := make([]*kycv1.Application, 0, len(apps))
	for _, app := range apps {
		items = append(items, applicationToProto(app))
	}

	return &kycv1.ListApplicationsResponse{Applications: items}, nil
}

func (s *KycServer) ClaimApplication(ctx context.Context, in *kycv1.ClaimApplicationRequest) (*kycv1.Application, error) {
	id, officerID, err := s.parseReview(in.ApplicationId, in.OfficerId)
	if err != nil {
		return nil, err
	}

	app, err := s.service.Application.Claim(ctx, id, officerID)
	if err != nil {
		return nil, s.fail("failed to claim application", err, domain.ErrReviewFailed, "application_id", in.ApplicationId)
	}

	return applicationToProto(app), nil
}

func (s *KycServer) ApproveApplication(ctx context.Context, in *kycv1.ApproveApplicationRequest) (*kycv1.Application, error) {
	id, officerID, err := s.parseReview(in.ApplicationId, in.OfficerId)
	if err != nil {
		return nil, err
	}

	app, err := s.service.Application.Approve(ctx, id, officerID, in.Comment)
	if err != nil {
		return nil, s.fail("failed to approve application", err, domain.ErrReviewFailed, "application_id", in.ApplicationId)
	}

	return applicationToProto(app), nil
}

func (s *KycServer) RejectApplication(ctx context.Context, in *kycv1.RejectApplicationRequest) (*kycv1.Application, error) {
	id, officerID, err := s.parseReview(in.ApplicationId, in.OfficerId)
	if err != nil {
		return nil, err
	}

	app, err := s.service.Application.Reject(ctx, id, officerID, in.Reason)
	if err != nil {
		return nil, s.fail("failed to reject application", err, domain.ErrReviewFailed, "application_id", in.ApplicationId)
	}

	return applicationToProto(app), nil
}

func (s *KycServer) RequestMoreDocuments(ctx context.Context, in *kycv1.RequestMoreDocumentsRequest) (*kycv1.Application, error) {
	id, officerID, err := s.parseReview(in.ApplicationId, in.OfficerId)
	if err != nil {
		return nil, err
	}

	app, err := s.service.Application.RequestDocuments(ctx, id, officerID, documentTypesFromProto(in.Types), in.Comment)
	if err != nil {
		return nil, s.fail("failed to request documents", err, domain.ErrReviewFailed, "application_id", in.ApplicationId)
	}

	return applicationToProto(app), nil
}

// fail переводит доменную ошибку в статус наружу. Всё, что не разобрано,
// уходит как fallback, а причина остаётся в логе.
func (s *KycServer) AddSanctionEntry(ctx context.Context, in *kycv1.AddSanctionEntryRequest) (*kycv1.SanctionEntry, error) {
	entry, err := s.service.Sanctions.Add(ctx, models.SanctionEntry{
		FirstName: in.FirstName,
		LastName:  in.LastName,
		BirthDate: dateFromProto(in.BirthDate),
		Source:    in.Source,
	})
	if err != nil {
		return nil, s.fail("failed to add sanction entry", err, domain.ErrAddSanctionFailed)
	}

	return sanctionEntryToProto(entry), nil
}

func (s *KycServer) ListSanctions(ctx context.Context, in *kycv1.ListSanctionsRequest) (*kycv1.ListSanctionsResponse, error) {
	entries, err := s.service.Sanctions.List(ctx, int(in.Limit), int(in.Offset))
	if err != nil {
		return nil, s.fail("failed to list sanction entries", err, domain.ErrListSanctionsFailed)
	}

	return &kycv1.ListSanctionsResponse{Entries: sanctionEntriesToProto(entries)}, nil
}

func (s *KycServer) fail(msg string, err error, fallback error, logArgs ...any) error {
	switch {
	case errors.Is(err, domain.ErrApplicationNotFound):
		return domain.ErrApplicationMissing
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrCustomerMissing
	case errors.Is(err, domain.ErrActiveApplicationExists):
		return domain.ErrApplicationExists
	case errors.Is(err, domain.ErrProfileNotFilled):
		return domain.ErrProfileIsNotFilled
	case errors.Is(err, domain.ErrDocumentsMissing):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrStatusConflict):
		return domain.ErrTransitionForbidden
	case errors.Is(err, domain.ErrAlreadyClaimed):
		return domain.ErrClaimedByOther
	case errors.Is(err, domain.ErrReasonRequired):
		return domain.ErrReasonIsRequired
	case errors.Is(err, domain.ErrDocumentTypesRequired):
		return domain.ErrTypesRequired
	case errors.Is(err, domain.ErrSanctionEntryExists):
		return domain.ErrSanctionExists
	}

	if _, isStatus := status.FromError(err); isStatus {
		return err
	}

	s.log.Error(msg, append(logArgs, "error", err)...)

	return fallback
}

func (s *KycServer) parseReview(applicationID, officerID string) (uuid.UUID, uuid.UUID, error) {
	id, err := uuid.Parse(applicationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.ErrApplicationIDMustBeUUID
	}

	officer, err := uuid.Parse(officerID)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.ErrOfficerIDMustBeUUID
	}

	return id, officer, nil
}
