package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// Runner прогоняет авто-проверки по заявке. Каждая проверка - отдельный метод,
// результат одинаковой формы: тип, вердикт и подробности для аудита.
type Runner struct {
	sanctions interfaces.SanctionsRepository
}

func NewRunner(sanctions interfaces.SanctionsRepository) *Runner {
	return &Runner{sanctions: sanctions}
}

// Run прогоняет все проверки из models.AllCheckTypes. Заявка проходит их
// целиком, а не до первого отказа: офицеру и аудиту нужна полная картина.
func (r *Runner) Run(ctx context.Context, app *models.Application) ([]models.Check, error) {
	results := make([]models.Check, 0, len(models.AllCheckTypes()))

	for _, checkType := range models.AllCheckTypes() {
		var (
			check models.Check
			err   error
		)

		switch checkType {
		case models.CHECK_TYPE_SANCTIONS:
			check, err = r.sanctionsCheck(ctx, app)
		case models.CHECK_TYPE_AGE:
			check = ageCheck(app)
		default:
			return nil, fmt.Errorf("проверка %q не реализована", checkType)
		}

		if err != nil {
			return nil, err
		}

		check.ApplicationID = app.ID
		check.Type = checkType

		results = append(results, check)
	}

	return results, nil
}

// sanctionsCheck сверяет снимок анкеты с мок-списком. Сам список лежит в базе
// сервиса; настоящий приезжал бы из внешнего источника.
func (r *Runner) sanctionsCheck(ctx context.Context, app *models.Application) (models.Check, error) {
	entry, err := r.sanctions.FindMatch(ctx, app.FirstName, app.LastName, birthDate(app))
	if err != nil {
		return models.Check{}, err
	}

	if entry == nil {
		return models.Check{Result: models.CHECK_RESULT_PASSED, Details: "no match"}, nil
	}

	return models.Check{
		Result:  models.CHECK_RESULT_FAILED,
		Details: fmt.Sprintf("match in %s: %s %s", entry.Source, entry.FirstName, entry.LastName),
	}, nil
}

// ageCheck считает возраст на сегодня. Дата рождения - снимок на момент
// подачи, поэтому результат не поедет от правки анкеты.
func ageCheck(app *models.Application) models.Check {
	if app.BirthDate == nil {
		return models.Check{Result: models.CHECK_RESULT_FAILED, Details: "birth date is missing"}
	}

	age := fullYears(*app.BirthDate, time.Now().UTC())

	if age < domain.MIN_CUSTOMER_AGE_YEARS {
		return models.Check{
			Result:  models.CHECK_RESULT_FAILED,
			Details: fmt.Sprintf("age %d is under %d", age, domain.MIN_CUSTOMER_AGE_YEARS),
		}
	}

	return models.Check{
		Result:  models.CHECK_RESULT_PASSED,
		Details: fmt.Sprintf("age %d", age),
	}
}

// fullYears считает полные годы: день рождения в этом году ещё не наступил -
// год не засчитывается.
func fullYears(birth, now time.Time) int {
	years := now.Year() - birth.Year()

	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}

	return years
}

func birthDate(app *models.Application) time.Time {
	if app.BirthDate == nil {
		return time.Time{}
	}

	return *app.BirthDate
}
