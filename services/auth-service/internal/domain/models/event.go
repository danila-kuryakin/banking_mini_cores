package models

import "time"

type UserRegistered struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	RegisteredAt time.Time `json:"registered_at"`
}
