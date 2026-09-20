package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
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

// Add добавляет человека в санкционный список; такой же человек уже в списке - ErrSanctionEntryExists.
func (r *SanctionsRepo) Add(ctx context.Context, entry models.SanctionEntry) (*models.SanctionEntry, error) {
	var added models.SanctionEntry

	err := r.db.QueryRow(ctx, INSERT_SANCTION_QUERY,
		entry.FirstName,
		entry.LastName,
		entry.BirthDate,
		entry.Source,
	).Scan(
		&added.ID,
		&added.FirstName,
		&added.LastName,
		&added.BirthDate,
		&added.Source,
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, domain.ErrSanctionEntryExists
		}

		return nil, fmt.Errorf("add sanction: %w", err)
	}

	return &added, nil
}

// List отдаёт санкционный список постранично.
func (r *SanctionsRepo) List(ctx context.Context, limit, offset int) ([]models.SanctionEntry, error) {
	rows, err := r.db.Query(ctx, LIST_SANCTIONS_QUERY, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list sanctions: %w", err)
	}
	defer rows.Close()

	entries := make([]models.SanctionEntry, 0, limit)

	for rows.Next() {
		var entry models.SanctionEntry

		err := rows.Scan(
			&entry.ID,
			&entry.FirstName,
			&entry.LastName,
			&entry.BirthDate,
			&entry.Source,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sanctions: %w", err)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sanctions: %w", err)
	}

	return entries, nil
}
