package models

import (
	"time"

	"github.com/google/uuid"
)

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

type Customer struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          Status
	StatusChangedAt time.Time
	Profile         Profile
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
