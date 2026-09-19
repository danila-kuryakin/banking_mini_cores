package models

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// CheckType и CheckResult совпадают с метками enum check_type и check_result.
type CheckType string

const (
	CHECK_TYPE_UNSPECIFIED CheckType = ""
	CHECK_TYPE_SANCTIONS   CheckType = "sanctions"
	CHECK_TYPE_AGE         CheckType = "age"
)

type CheckResult string

const (
	CHECK_RESULT_UNSPECIFIED CheckResult = ""
	CHECK_RESULT_PASSED      CheckResult = "passed"
	CHECK_RESULT_FAILED      CheckResult = "failed"
)

var allCheckTypes = []CheckType{
	CHECK_TYPE_SANCTIONS,
	CHECK_TYPE_AGE,
}

// AllCheckTypes - полный набор авто-проверок. Заявка проходит их все.
func AllCheckTypes() []CheckType {
	return slices.Clone(allCheckTypes)
}

// Check - результат одной авто-проверки по заявке.
type Check struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	Type          CheckType
	Result        CheckResult
	Details       string
	CreatedAt     time.Time
}

// Passed сообщает, что проверка пройдена.
func (c Check) Passed() bool {
	return c.Result == CHECK_RESULT_PASSED
}

// SanctionEntry - совпадение в санкционном списке.
type SanctionEntry struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	BirthDate time.Time
	Source    string
}
