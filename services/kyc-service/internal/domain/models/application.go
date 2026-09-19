package models

import (
	"time"

	"github.com/google/uuid"
)

// Application - заявка на верификацию.
//
// FirstName, LastName и BirthDate - снимок анкеты на момент подачи: анкета
// живёт в customer-service и может измениться, а решение по заявке принималось
// по этим данным.
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
//
// Reason и RequestedTypes перезаписываются на каждом переходе: повторная
// подача после docs_requested обязана очистить и причину, и список типов.
type Transition struct {
	To             ApplicationStatus
	ActorID        *uuid.UUID
	Comment        string
	Reason         string
	RequestedTypes []DocumentType
	// Decided проставляет decided_at: переход стал решением по заявке.
	Decided bool
}

// ApplicationFilter - фильтры очереди офицера. Пустой Statuses - все статусы.
type ApplicationFilter struct {
	Statuses   []ApplicationStatus
	ReviewerID *uuid.UUID
	Limit      int
	Offset     int
}
