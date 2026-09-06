package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	STATUS_NEW            Status = "new"
	STATUS_PROFILE_FILLED Status = "profile_filled"
	STATUS_ON_KYC         Status = "on_kyc"
	STATUS_ACTIVE         Status = "active"
	STATUS_REJECTED       Status = "rejected"
	STATUS_BLOCKED        Status = "blocked"
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
