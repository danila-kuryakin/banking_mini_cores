package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// row - Общий тип для сканирования.
type row interface {
	Scan(dest ...any) error
}

// scanApplication - сканирование одной строки
func scanApplication(r row) (*models.Application, error) {
	var (
		app            models.Application
		status         string
		requestedTypes []string
	)

	err := r.Scan(
		&app.ID,
		&app.UserID,
		&status,
		&app.Reviewer,
		&app.Reason,
		&requestedTypes,
		&app.FirstName,
		&app.LastName,
		&app.BirthDate,
		&app.SubmittedAt,
		&app.DecidedAt,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	app.Status = models.ApplicationStatus(status)
	app.RequestedTypes = documentTypes(requestedTypes)

	return &app, nil
}

// collectApplications - сканирование нескольких строк
func collectApplications(rows pgx.Rows, capacity int) ([]*models.Application, error) {
	if capacity < 0 {
		capacity = 0
	}

	apps := make([]*models.Application, 0, capacity)

	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, fmt.Errorf("scan applications: %w", err)
		}

		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applications: %w", err)
	}

	return apps, nil
}

// SybQuery - Общий тип для промежуточных запросов.
type SybQuery interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// listEvents Под запрос, возвращающий список переходов статусов (событий).
func listEvents(ctx context.Context, q SybQuery, applicationID uuid.UUID) ([]models.Event, error) {
	rows, err := q.Query(ctx, LIST_EVENTS_QUERY, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := make([]models.Event, 0)

	for rows.Next() {
		var (
			event    models.Event
			from     *string
			toStatus string
		)

		err := rows.Scan(
			&event.ID,
			&event.ApplicationID,
			&from,
			&toStatus,
			&event.ActorID,
			&event.Comment,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan events: %w", err)
		}

		if from != nil {
			event.From = models.ApplicationStatus(*from)
		}

		event.To = models.ApplicationStatus(toStatus)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

// listChecks Под запрос, возвращающий список авто-проверок по одной заявке.
func listChecks(ctx context.Context, q SybQuery, applicationID uuid.UUID) ([]models.Check, error) {
	rows, err := q.Query(ctx, LIST_CHECKS_QUERY, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	defer rows.Close()

	checks := make([]models.Check, 0)

	for rows.Next() {
		check, err := scanCheck(rows)
		if err != nil {
			return nil, err
		}

		checks = append(checks, *check)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checks: %w", err)
	}

	return checks, nil
}

// scanCheck Сканирование статуса.
func scanCheck(r row) (*models.Check, error) {
	var (
		check     models.Check
		checkType string
		result    string
		createdAt time.Time
	)

	err := r.Scan(
		&check.ID,
		&check.ApplicationID,
		&checkType,
		&result,
		&check.Details,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan check: %w", err)
	}

	check.Type = models.CheckType(checkType)
	check.Result = models.CheckResult(result)
	check.CreatedAt = createdAt

	return &check, nil
}

// statusStrings переводит доменные списки в срезы строк.
func statusStrings(in []models.ApplicationStatus) []string {
	out := make([]string, 0, len(in))

	for _, status := range in {
		out = append(out, string(status))
	}

	return out
}

// documentTypeStrings переводит доменные списки в срезы строк.
func documentTypeStrings(in []models.DocumentType) []string {
	out := make([]string, 0, len(in))

	for _, docType := range in {
		out = append(out, string(docType))
	}

	return out
}

// documentTypes переводит срезы строк в доменные списки.
func documentTypes(in []string) []models.DocumentType {
	out := make([]models.DocumentType, 0, len(in))

	for _, raw := range in {
		out = append(out, models.DocumentType(raw))
	}

	return out
}
