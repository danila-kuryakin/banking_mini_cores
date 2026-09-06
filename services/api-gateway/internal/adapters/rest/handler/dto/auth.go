package dto

import "time"

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type RegisterResponse struct {
	ID        string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}
type LoginResponse struct {
	User   User      `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ValidateTokenRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

type ValidateTokenResponse struct {
	Valid     bool      `json:"valid"`
	ID        string    `json:"id,omitempty"`
	Role      string    `json:"role,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitzero"`
}

type CreateOfficerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
