package dto

import (
	"time"
)

type UpdateProfile struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	BirthDate   string `json:"birth_date"`
	Citizenship string `json:"citizenship"`
	Phone       string `json:"phone"`
}

type Profile struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	BirthDate   string `json:"birth_date"`
	Citizenship string `json:"citizenship"`
	Phone       string `json:"phone"`
}

type Customer struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	Profile   Profile   `json:"profile"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomerStatus struct {
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListCustomersQuery struct {
	Status    string `json:"status"`
	PageSize  int32  `json:"page_size"`
	PageToken string `json:"page_token"`
}

type ListCustomers struct {
	Customers     []Customer `json:"customers"`
	NextPageToken string     `json:"next_page_token"`
	TotalSize     int32      `json:"total_size"`
}
