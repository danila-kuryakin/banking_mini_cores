package models

import (
	"time"

	"github.com/google/uuid"
)

// Event - запись аудита: переход заявки из одного статуса в другой.
type Event struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	From          ApplicationStatus
	To            ApplicationStatus
	ActorID       *uuid.UUID
	Comment       string
	CreatedAt     time.Time
}
