package models

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
)

var phonePattern = regexp.MustCompile(domain.PHONE_PATTERN)

type Profile struct {
	FirstName   string
	LastName    string
	BirthDate   *time.Time
	Citizenship string
	Phone       string
}

func (p Profile) IsComplete() bool {
	return p.FirstName != "" &&
		p.LastName != "" &&
		p.BirthDate != nil &&
		p.Citizenship != "" &&
		p.Phone != ""
}

// Normalize отдаёт копию анкеты без пробелов по краям строковых полей.
func (p Profile) Normalize() Profile {
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	p.Citizenship = strings.TrimSpace(p.Citizenship)
	p.Phone = strings.TrimSpace(p.Phone)

	return p
}

// Validate проверяет нормализованную анкету на момент now.
func (p Profile) Validate(now time.Time) error {
	switch {
	case utf8.RuneCountInString(p.FirstName) > domain.NAME_MAX_LENGTH:
		return domain.ErrFirstNameIsTooLong
	case utf8.RuneCountInString(p.LastName) > domain.NAME_MAX_LENGTH:
		return domain.ErrLastNameIsTooLong
	case utf8.RuneCountInString(p.Citizenship) > domain.CITIZENSHIP_MAX_LENGTH:
		return domain.ErrCitizenshipIsTooLong
	case p.Phone != "" && !phonePattern.MatchString(p.Phone):
		return domain.ErrPhoneIsNotValid
	}

	switch {
	case p.BirthDate == nil:
		return nil
	case p.BirthDate.After(now):
		return domain.ErrBirthDateInFuture
	case p.BirthDate.Before(now.AddDate(-domain.MAX_CUSTOMER_AGE_YEARS, 0, 0)):
		return domain.ErrBirthDateIsTooOld
	default:
		return nil
	}
}

type Customer struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          Status
	StatusChangedAt time.Time
	Profile         Profile
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
