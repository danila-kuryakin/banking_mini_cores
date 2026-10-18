package models

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status AccountStatus
		want   bool
	}{
		{name: "active", status: STATUS_ACTIVE, want: true},
		{name: "blocked", status: STATUS_BLOCKED, want: true},
		{name: "closed", status: STATUS_CLOSED, want: true},
		{name: "empty", status: AccountStatus(""), want: false},
		{name: "upper case", status: AccountStatus("ACTIVE"), want: false},
		{name: "unknown", status: AccountStatus("frozen"), want: false},
		{name: "leading space", status: AccountStatus(" active"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from AccountStatus
		to   AccountStatus
		want bool
	}{
		{name: "active to blocked", from: STATUS_ACTIVE, to: STATUS_BLOCKED, want: true},
		{name: "blocked to active", from: STATUS_BLOCKED, to: STATUS_ACTIVE, want: true},
		{name: "active to closed", from: STATUS_ACTIVE, to: STATUS_CLOSED, want: true},
		{name: "active to active", from: STATUS_ACTIVE, to: STATUS_ACTIVE, want: false},
		{name: "blocked to blocked", from: STATUS_BLOCKED, to: STATUS_BLOCKED, want: false},
		{name: "closed to closed", from: STATUS_CLOSED, to: STATUS_CLOSED, want: false},
		{name: "blocked to closed", from: STATUS_BLOCKED, to: STATUS_CLOSED, want: false},
		{name: "closed to active", from: STATUS_CLOSED, to: STATUS_ACTIVE, want: false},
		{name: "closed to blocked", from: STATUS_CLOSED, to: STATUS_BLOCKED, want: false},
		{name: "empty to active", from: AccountStatus(""), to: STATUS_ACTIVE, want: false},
		{name: "active to empty", from: STATUS_ACTIVE, to: AccountStatus(""), want: false},
		{name: "unknown to active", from: AccountStatus("frozen"), to: STATUS_ACTIVE, want: false},
		{name: "active to unknown", from: STATUS_ACTIVE, to: AccountStatus("frozen"), want: false},
		{name: "empty to empty", from: AccountStatus(""), to: AccountStatus(""), want: false},
		{name: "upper case to blocked", from: AccountStatus("ACTIVE"), to: STATUS_BLOCKED, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CanTransition(tt.from, tt.to))
		})
	}
}

func TestAllowedFrom(t *testing.T) {
	tests := []struct {
		name string
		to   AccountStatus
		want []AccountStatus
	}{
		{name: "active", to: STATUS_ACTIVE, want: []AccountStatus{STATUS_BLOCKED}},
		{name: "blocked", to: STATUS_BLOCKED, want: []AccountStatus{STATUS_ACTIVE}},
		{name: "closed", to: STATUS_CLOSED, want: []AccountStatus{STATUS_ACTIVE}},
		{name: "empty", to: AccountStatus(""), want: nil},
		{name: "unknown", to: AccountStatus("frozen"), want: nil},
		{name: "upper case", to: AccountStatus("ACTIVE"), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllowedFrom(tt.to)
			// assert.Equal различает nil и пустой срез через reflect.DeepEqual
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAllowedFrom_ReturnsCopy(t *testing.T) {
	got := AllowedFrom(STATUS_CLOSED)
	require.Len(t, got, 1)
	got[0] = STATUS_BLOCKED

	assert.Equal(t, []AccountStatus{STATUS_ACTIVE}, AllowedFrom(STATUS_CLOSED))
}

func TestCanTransition_ConsistentWithAllowedFrom(t *testing.T) {
	statuses := []AccountStatus{STATUS_ACTIVE, STATUS_BLOCKED, STATUS_CLOSED}

	for _, to := range statuses {
		for _, from := range statuses {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				assert.Equal(t, slices.Contains(AllowedFrom(to), from), CanTransition(from, to))
			})
		}
	}
}
