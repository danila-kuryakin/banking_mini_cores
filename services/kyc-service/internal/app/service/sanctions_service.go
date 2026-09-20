package service

import (
	"context"
	"strings"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type SanctionsService struct {
	repo *repository.Repository
}

func NewSanctionsService(repo *repository.Repository) *SanctionsService {
	return &SanctionsService{repo: repo}
}

// Add добавляет человека в список; пробелы по краям срезаются, иначе сверка по имени его не найдёт.
func (s *SanctionsService) Add(ctx context.Context, entry models.SanctionEntry) (*models.SanctionEntry, error) {
	entry.FirstName = strings.TrimSpace(entry.FirstName)
	entry.LastName = strings.TrimSpace(entry.LastName)
	entry.Source = strings.TrimSpace(entry.Source)

	return s.repo.Sanctions.Add(ctx, entry)
}

// List отдаёт санкционный список; размер страницы ограничен, как в очереди заявок.
func (s *SanctionsService) List(ctx context.Context, limit, offset int) ([]models.SanctionEntry, error) {
	if limit <= 0 {
		limit = domain.DEFAULT_PAGE_SIZE
	} else if limit > domain.MAX_PAGE_SIZE {
		limit = domain.MAX_PAGE_SIZE
	}

	if offset < 0 {
		offset = 0
	}

	return s.repo.Sanctions.List(ctx, limit, offset)
}
