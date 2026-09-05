package domain

import "time"

type ValidateTokenOutput struct {
	Valid     bool
	ID        string
	Role      string
	ExpiresAt time.Time
}
