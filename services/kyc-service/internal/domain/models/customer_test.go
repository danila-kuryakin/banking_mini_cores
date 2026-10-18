package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	TEST_FIRST_NAME = "Ivan"
	TEST_LAST_NAME  = "Petrov"
)

func TestCustomerProfile_IsComplete(t *testing.T) {
	birthDate := time.Date(1990, time.May, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		profile CustomerProfile
		want    bool
	}{
		{
			name:    "complete_profile_filled",
			profile: CustomerProfile{FirstName: TEST_FIRST_NAME, LastName: TEST_LAST_NAME, BirthDate: &birthDate, Status: CustomerStatus("profile_filled")},
			want:    true,
		},
		{
			name:    "complete_empty_status",
			profile: CustomerProfile{FirstName: TEST_FIRST_NAME, LastName: TEST_LAST_NAME, BirthDate: &birthDate, Status: CustomerStatus("")},
			want:    true,
		},
		{
			name:    "complete_rejected_status",
			profile: CustomerProfile{FirstName: TEST_FIRST_NAME, LastName: TEST_LAST_NAME, BirthDate: &birthDate, Status: CustomerStatus("rejected")},
			want:    true,
		},
		{
			name:    "empty_first_name",
			profile: CustomerProfile{FirstName: "", LastName: TEST_LAST_NAME, BirthDate: &birthDate},
			want:    false,
		},
		{
			name:    "empty_last_name",
			profile: CustomerProfile{FirstName: TEST_FIRST_NAME, LastName: "", BirthDate: &birthDate},
			want:    false,
		},
		{
			name:    "nil_birth_date",
			profile: CustomerProfile{FirstName: TEST_FIRST_NAME, LastName: TEST_LAST_NAME, BirthDate: nil},
			want:    false,
		},
		{
			name:    "zero_profile",
			profile: CustomerProfile{},
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.profile.IsComplete())
		})
	}
}
