package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

func (r Role) String() string {
	return string(r)
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
