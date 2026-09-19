package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type ApplicationService struct {
	repo    *repository.Repository
	clients *client.Clients
}

func NewApplicationService(repo *repository.Repository, clients *client.Clients) *ApplicationService {
	return &ApplicationService{repo: repo, clients: clients}
}

// Submit подаёт заявку.
//
// Два пути: заявка, по которой запросили документы, возвращается в submitted
// той же строкой, всё остальное - новая заявка. Различать их обязательно:
// клиент с висящей заявкой находится в on_kyc, а не в profile_filled.
func (s *ApplicationService) Submit(ctx context.Context, userID uuid.UUID) (*models.Application, error) {
	latest, err := s.repo.Application.GetLatestByUserID(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrApplicationNotFound) {
		return nil, err
	}

	if latest != nil && latest.Status == models.STATUS_DOCS_REQUESTED {
		return s.resubmit(ctx, userID, latest)
	}

	if latest != nil && latest.Status.IsActive() {
		return nil, domain.ErrActiveApplicationExists
	}

	profile, err := s.clients.Customer.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	if profile.Status != models.CUSTOMER_STATUS_PROFILE_FILLED || !profile.IsComplete() {
		return nil, domain.ErrProfileNotFilled
	}

	if err := s.requireDocuments(ctx, userID, nil); err != nil {
		return nil, err
	}

	created, err := s.repo.Application.Create(ctx, models.Application{
		UserID:    userID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		BirthDate: profile.BirthDate,
	})
	if err != nil {
		return nil, err
	}

	err = s.clients.Customer.SetStatus(ctx, userID, models.CUSTOMER_STATUS_ON_KYC, domain.REASON_APPLICATION_SUBMITTED, nil)
	if err != nil {
		return nil, fmt.Errorf("submit application: %w", err)
	}

	return created, nil
}

// resubmit возвращает в очередь заявку, по которой запрашивали документы.
// Статус клиента не трогаем: он всё это время оставался в on_kyc.
func (s *ApplicationService) resubmit(ctx context.Context, userID uuid.UUID, app *models.Application) (*models.Application, error) {
	if err := s.requireDocuments(ctx, userID, app.RequestedTypes); err != nil {
		return nil, err
	}

	return s.transition(ctx, app.ID, models.Transition{To: models.STATUS_SUBMITTED})
}

// requireDocuments проверяет комплектность: обязательные типы спрашиваем у
// filestore, а дополнительно запрошенные офицером сверяем сами - в его набор
// обязательных они не входят.
func (s *ApplicationService) requireDocuments(ctx context.Context, userID uuid.UUID, requested []models.DocumentType) error {
	ok, missing, err := s.clients.Documents.HasRequiredDocuments(ctx, userID)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrDocumentsMissing, joinTypes(missing))
	}

	if len(requested) == 0 {
		return nil
	}

	confirmed, err := s.clients.Documents.ConfirmedTypes(ctx, userID)
	if err != nil {
		return err
	}

	stillMissing := make([]models.DocumentType, 0, len(requested))

	for _, docType := range requested {
		if !slices.Contains(confirmed, docType) {
			stillMissing = append(stillMissing, docType)
		}
	}

	if len(stillMissing) > 0 {
		return fmt.Errorf("%w: %s", domain.ErrDocumentsMissing, joinTypes(stillMissing))
	}

	return nil
}

func joinTypes(types []models.DocumentType) string {
	parts := make([]string, 0, len(types))

	for _, docType := range types {
		parts = append(parts, string(docType))
	}

	return strings.Join(parts, ", ")
}

func (s *ApplicationService) GetByID(ctx context.Context, id uuid.UUID) (*models.Application, error) {
	return s.repo.Application.GetByID(ctx, id)
}

func (s *ApplicationService) GetLatestByUserID(ctx context.Context, userID uuid.UUID) (*models.Application, error) {
	return s.repo.Application.GetLatestByUserID(ctx, userID)
}

func (s *ApplicationService) List(ctx context.Context, filter models.ApplicationFilter) ([]*models.Application, error) {
	if filter.Limit <= 0 {
		filter.Limit = domain.DEFAULT_PAGE_SIZE
	} else if filter.Limit > domain.MAX_PAGE_SIZE {
		filter.Limit = domain.MAX_PAGE_SIZE
	}

	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repo.Application.List(ctx, filter)
}

// Claim закрепляет заявку за офицером. Статуса не меняет: после авто-проверок
// заявка уже в in_review, и очередь - это заявки без reviewer_id.
func (s *ApplicationService) Claim(ctx context.Context, id, officerID uuid.UUID) (*models.Application, error) {
	return s.repo.Application.Claim(ctx, id, officerID)
}

// Approve одобряет заявку и переводит клиента в active.
func (s *ApplicationService) Approve(ctx context.Context, id, officerID uuid.UUID, comment string) (*models.Application, error) {
	app, err := s.transition(ctx, id, models.Transition{
		To:      models.STATUS_APPROVED,
		ActorID: &officerID,
		Comment: comment,
		Decided: true,
	})
	if err != nil {
		return nil, err
	}

	err = s.clients.Customer.SetStatus(ctx, app.UserID, models.CUSTOMER_STATUS_ACTIVE, domain.REASON_APPLICATION_APPROVED, &officerID)
	if err != nil {
		return nil, fmt.Errorf("approve application: %w", err)
	}

	return app, nil
}

// Reject отказывает по заявке. Клиент уходит в rejected окончательно - в
// отличие от авто-отказа, который возвращает его в profile_filled.
func (s *ApplicationService) Reject(ctx context.Context, id, officerID uuid.UUID, reason string) (*models.Application, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, domain.ErrReasonRequired
	}

	app, err := s.transition(ctx, id, models.Transition{
		To:      models.STATUS_REJECTED,
		ActorID: &officerID,
		Comment: reason,
		Reason:  reason,
		Decided: true,
	})
	if err != nil {
		return nil, err
	}

	err = s.clients.Customer.SetStatus(ctx, app.UserID, models.CUSTOMER_STATUS_REJECTED, reason, &officerID)
	if err != nil {
		return nil, fmt.Errorf("reject application: %w", err)
	}

	return app, nil
}

// RequestDocuments возвращает заявку клиенту за документами. Клиент остаётся в
// on_kyc: заявка жива, просто ждёт файлов.
func (s *ApplicationService) RequestDocuments(
	ctx context.Context,
	id, officerID uuid.UUID,
	types []models.DocumentType,
	comment string,
) (*models.Application, error) {
	if len(types) == 0 {
		return nil, domain.ErrDocumentTypesRequired
	}

	for _, docType := range types {
		if !docType.IsValid() {
			return nil, fmt.Errorf("%w: %s", domain.ErrDocumentTypesRequired, docType)
		}
	}

	return s.transition(ctx, id, models.Transition{
		To:             models.STATUS_DOCS_REQUESTED,
		ActorID:        &officerID,
		Comment:        comment,
		Reason:         comment,
		RequestedTypes: types,
	})
}

// transition - смена статуса заявки.
func (s *ApplicationService) transition(ctx context.Context, id uuid.UUID, t models.Transition) (*models.Application, error) {
	return s.repo.Application.Transition(ctx, id, t, models.AllowedFrom(t.To))
}
