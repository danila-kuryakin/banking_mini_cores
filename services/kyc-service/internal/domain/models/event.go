package models

import (
	"time"

	"github.com/google/uuid"
)

// Event - запись аудита: переход заявки из одного статуса в другой.
//
// From равен STATUS_UNSPECIFIED у записи о создании заявки: исходного статуса
// не было. ActorID пуст у системных переходов - их делает воркер проверок.
type Event struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	From          ApplicationStatus
	To            ApplicationStatus
	ActorID       *uuid.UUID
	Comment       string
	CreatedAt     time.Time
}
