package checks

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

const (
	TEST_FIRST_NAME          = "Ivan"
	TEST_LAST_NAME           = "Petrov"
	TEST_SANCTION_FIRST_NAME = "IVAN"
	TEST_SANCTION_LAST_NAME  = "PETROV"
	TEST_SANCTION_SOURCE     = "OFAC"
)

var (
	TEST_APPLICATION_ID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_APPLICATION_ID_2 = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TEST_APPLICATION_ID_3 = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	TEST_APPLICATION_ID_4 = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	TEST_USER_ID          = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

// date - полночь UTC указанного дня
func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// testToday - сегодняшняя полночь UTC; 29 февраля заменяется на 28-е, чтобы AddDate не нормализовал дату
func testToday() time.Time {
	now := time.Now().UTC()
	today := date(now.Year(), now.Month(), now.Day())
	if today.Month() == time.February && today.Day() == 29 {
		return today.AddDate(0, 0, -1)
	}
	return today
}

// birthDateAgo - дата рождения years лет назад со сдвигом days дней от сегодняшней
func birthDateAgo(years, days int) *time.Time {
	birthDate := testToday().AddDate(-years, 0, days)
	return &birthDate
}

// testSanctionEntry - запись санкционного списка для совпадения
func testSanctionEntry() *models.SanctionEntry {
	return &models.SanctionEntry{
		ID:        uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		FirstName: TEST_SANCTION_FIRST_NAME,
		LastName:  TEST_SANCTION_LAST_NAME,
		BirthDate: date(1990, time.January, 1),
		Source:    TEST_SANCTION_SOURCE,
	}
}

// testApplication - заявка со снимком анкеты
func testApplication(applicationID uuid.UUID, birthDate *time.Time) *models.Application {
	return &models.Application{
		ID:        applicationID,
		UserID:    TEST_USER_ID,
		FirstName: TEST_FIRST_NAME,
		LastName:  TEST_LAST_NAME,
		BirthDate: birthDate,
	}
}

// wantSanctionsCheck - ожидаемый результат санкционной проверки
func wantSanctionsCheck(applicationID uuid.UUID, result models.CheckResult, details string) models.Check {
	return models.Check{ApplicationID: applicationID, Type: models.CHECK_TYPE_SANCTIONS, Result: result, Details: details}
}

// wantAgeCheck - ожидаемый результат проверки возраста
func wantAgeCheck(applicationID uuid.UUID, result models.CheckResult, details string) models.Check {
	return models.Check{ApplicationID: applicationID, Type: models.CHECK_TYPE_AGE, Result: result, Details: details}
}

func TestFullYears(t *testing.T) {
	tests := []struct {
		name  string
		birth time.Time
		now   time.Time
		want  int
	}{
		{name: "day before birthday", birth: date(2000, time.May, 10), now: date(2018, time.May, 9), want: 17},
		{name: "exactly birthday", birth: date(2000, time.May, 10), now: date(2018, time.May, 10), want: 18},
		{name: "day after birthday", birth: date(2000, time.May, 10), now: date(2018, time.May, 11), want: 18},
		// 2018-2000=18, но 31 декабря ещё не наступило
		{name: "birthday month not reached", birth: date(2000, time.December, 31), now: date(2018, time.January, 1), want: 17},
		{name: "end of year after january birthday", birth: date(2000, time.January, 1), now: date(2018, time.December, 31), want: 18},
		{name: "same day as birth", birth: date(2000, time.May, 10), now: date(2000, time.May, 10), want: 0},
		{name: "february 28 birthday", birth: date(2000, time.February, 28), now: date(2018, time.February, 28), want: 18},
		{name: "leap birth day before february 28", birth: date(2000, time.February, 29), now: date(2018, time.February, 27), want: 17},
		{name: "leap birth march 1 non-leap year", birth: date(2000, time.February, 29), now: date(2018, time.March, 1), want: 18},
		// 2020 високосный, 29 февраля ещё впереди
		{name: "leap birth february 28 leap year", birth: date(2000, time.February, 29), now: date(2020, time.February, 28), want: 19},
		{name: "leap birth february 29 leap year", birth: date(2000, time.February, 29), now: date(2020, time.February, 29), want: 20},
		// тот же день месяца, но месяц ещё не наступил
		{name: "same day earlier month", birth: date(2000, time.May, 10), now: date(2018, time.April, 10), want: 17},
		// месяц уже прошёл, хотя число меньше
		{name: "later month smaller day", birth: date(2000, time.May, 10), now: date(2018, time.June, 9), want: 18},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fullYears(tt.birth, tt.now))
		})
	}
}

func TestRunner_Run(t *testing.T) {
	errFindMatch := errors.New("find match failed")
	otherEntry := &models.SanctionEntry{
		ID:        uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		FirstName: "Ioann",
		LastName:  "Petroff",
		BirthDate: date(1990, time.January, 1),
		Source:    "EU",
	}

	tests := []struct {
		name         string
		birthDate    *time.Time
		match        *models.SanctionEntry
		findMatchErr error
		want         []models.Check
		wantErr      error
	}{
		{
			name:      "adult not in list",
			birthDate: birthDateAgo(40, 0),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "no match"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "age 40"),
			},
		},
		{
			name:      "adult in list",
			birthDate: birthDateAgo(30, 0),
			match:     testSanctionEntry(),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "match in OFAC: IVAN PETROV"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "age 30"),
			},
		},
		{
			name:      "details taken from list entry not application",
			birthDate: birthDateAgo(30, 0),
			match:     otherEntry,
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "match in EU: Ioann Petroff"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "age 30"),
			},
		},
		{
			name:      "child in list gets both failures",
			birthDate: birthDateAgo(10, 0),
			match:     testSanctionEntry(),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "match in OFAC: IVAN PETROV"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "age 10 is under 18"),
			},
		},
		{
			name:      "turns 18 today",
			birthDate: birthDateAgo(18, 0),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "no match"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "age 18"),
			},
		},
		{
			name:      "turns 18 tomorrow",
			birthDate: birthDateAgo(18, 1),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "no match"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "age 17 is under 18"),
			},
		},
		{
			name:      "born today",
			birthDate: birthDateAgo(0, 0),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "no match"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "age 0 is under 18"),
			},
		},
		{
			name:      "missing birth date not in list",
			birthDate: nil,
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_PASSED, "no match"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "birth date is missing"),
			},
		},
		{
			name:      "missing birth date in list",
			birthDate: nil,
			match:     testSanctionEntry(),
			want: []models.Check{
				wantSanctionsCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "match in OFAC: IVAN PETROV"),
				wantAgeCheck(TEST_APPLICATION_ID, models.CHECK_RESULT_FAILED, "birth date is missing"),
			},
		},
		{
			name:         "find match error for adult",
			birthDate:    birthDateAgo(40, 0),
			findMatchErr: errFindMatch,
			wantErr:      errFindMatch,
		},
		{
			name:         "find match error for child",
			birthDate:    birthDateAgo(10, 0),
			findMatchErr: errFindMatch,
			wantErr:      errFindMatch,
		},
		{
			name:         "find match error without birth date",
			birthDate:    nil,
			findMatchErr: errFindMatch,
			wantErr:      errFindMatch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanctions := &fakeSanctionsRepository{match: tt.match, findMatchErr: tt.findMatchErr}
			runner := NewRunner(sanctions)

			got, err := runner.Run(t.Context(), testApplication(TEST_APPLICATION_ID, tt.birthDate))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunner_Run_FindMatchCalls(t *testing.T) {
	birthDate := birthDateAgo(30, 0)

	tests := []struct {
		name         string
		birthDate    *time.Time
		findMatchErr error
		want         []findMatchCall
	}{
		{
			name:      "birth date from snapshot",
			birthDate: birthDate,
			want:      []findMatchCall{{firstName: TEST_FIRST_NAME, lastName: TEST_LAST_NAME, birthDate: *birthDate}},
		},
		{
			name:      "missing birth date passed as zero time",
			birthDate: nil,
			want:      []findMatchCall{{firstName: TEST_FIRST_NAME, lastName: TEST_LAST_NAME, birthDate: time.Time{}}},
		},
		{
			name:         "single call on error",
			birthDate:    birthDate,
			findMatchErr: errors.New("find match failed"),
			want:         []findMatchCall{{firstName: TEST_FIRST_NAME, lastName: TEST_LAST_NAME, birthDate: *birthDate}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanctions := &fakeSanctionsRepository{findMatchErr: tt.findMatchErr}
			runner := NewRunner(sanctions)

			_, err := runner.Run(t.Context(), testApplication(TEST_APPLICATION_ID, tt.birthDate))
			require.ErrorIs(t, err, tt.findMatchErr)

			assert.Equal(t, tt.want, sanctions.calls())
		})
	}
}

func TestRunner_Run_ResultShape(t *testing.T) {
	tests := []struct {
		name          string
		applicationID uuid.UUID
		birthDate     *time.Time
		match         *models.SanctionEntry
	}{
		{name: "adult not in list", applicationID: TEST_APPLICATION_ID, birthDate: birthDateAgo(40, 0)},
		{name: "minor not in list", applicationID: TEST_APPLICATION_ID_2, birthDate: birthDateAgo(10, 0)},
		{name: "missing birth date", applicationID: TEST_APPLICATION_ID_3, birthDate: nil},
		{name: "adult in list", applicationID: TEST_APPLICATION_ID_4, birthDate: birthDateAgo(30, 0), match: testSanctionEntry()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanctions := &fakeSanctionsRepository{match: tt.match}
			runner := NewRunner(sanctions)

			got, err := runner.Run(t.Context(), testApplication(tt.applicationID, tt.birthDate))
			require.NoError(t, err)

			// порядок проверок задан R1: сначала санкции, потом возраст
			checkTypes := []models.CheckType{models.CHECK_TYPE_SANCTIONS, models.CHECK_TYPE_AGE}
			require.Len(t, got, len(checkTypes))
			for i, check := range got {
				assert.Equal(t, checkTypes[i], check.Type)
				assert.Equal(t, tt.applicationID, check.ApplicationID)
				assert.Equal(t, uuid.Nil, check.ID)
				assert.Equal(t, time.Time{}, check.CreatedAt)
			}
		})
	}
}
