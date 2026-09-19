package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type SanctionsRepo struct {
	db *pgxpool.Pool
}

func NewSanctionsRepo(db *pgxpool.Pool) *SanctionsRepo {
	return &SanctionsRepo{db: db}
}

// FindMatch возвращает nil без ошибки, если человека в списке нет: отсутствие
// совпадения - штатный исход проверки, а не сбой.
func (r *SanctionsRepo) FindMatch(ctx context.Context, firstName, lastName string, birthDate time.Time) (*models.SanctionEntry, error) {
	var entry models.SanctionEntry

	err := r.db.QueryRow(ctx, FIND_SANCTION_QUERY, lastName, firstName, birthDate).Scan(
		&entry.ID,
		&entry.FirstName,
		&entry.LastName,
		&entry.BirthDate,
		&entry.Source,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("find sanction: %w", err)
	}

	return &entry, nil
}
