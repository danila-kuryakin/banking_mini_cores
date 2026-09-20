package models

import (
	"time"

	"github.com/google/uuid"
)

// Application - заявка на верификацию.
type Application struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Status   ApplicationStatus
	Reviewer *uuid.UUID
	Reason   string

	RequestedTypes []DocumentType

	FirstName string
	LastName  string
	BirthDate *time.Time

	SubmittedAt time.Time
	DecidedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Заполняются только при чтении одной заявки.
	Checks []Check
	Events []Event
}

// Transition - смена статуса заявки с комментариями почему изменен статус.
type Transition struct {
	To             ApplicationStatus
	ActorID        *uuid.UUID
	Comment        string
	Reason         string
	RequestedTypes []DocumentType
	Decided        bool
}

// ApplicationFilter - фильтры очереди офицера. Пустой Statuses - все статусы.
type ApplicationFilter struct {
	Statuses   []ApplicationStatus
	ReviewerID *uuid.UUID
	Unassigned bool
	Limit      int
	Offset     int
}
