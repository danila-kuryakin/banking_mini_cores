package models

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allowedPairs - единственные разрешённые переходы, кроме перехода в blocked.
var allowedPairs = map[[2]Status]bool{
	{STATUS_NEW, STATUS_PROFILE_FILLED}:    true,
	{STATUS_ON_KYC, STATUS_PROFILE_FILLED}: true,
	{STATUS_PROFILE_FILLED, STATUS_ON_KYC}: true,
	{STATUS_ON_KYC, STATUS_ACTIVE}:         true,
	{STATUS_ON_KYC, STATUS_REJECTED}:       true,
}

func TestCanTransition_Matrix(t *testing.T) {
	statuses := append(AllStatuses(), STATUS_UNSPECIFIED)

	for _, from := range statuses {
		for _, to := range statuses {
			want := allowedPairs[[2]Status{from, to}]
			if to == STATUS_BLOCKED && from != STATUS_BLOCKED && from != STATUS_UNSPECIFIED {
				want = true
			}

			t.Run(string(from)+" -> "+string(to), func(t *testing.T) {
				assert.Equal(t, want, CanTransition(from, to), "unexpected result for %q -> %q", from, to)
			})
		}
	}
}

func TestCanTransition_UnknownStatus(t *testing.T) {
	unknown := Status("foo")

	for _, status := range append(AllStatuses(), STATUS_UNSPECIFIED) {
		t.Run(string(status), func(t *testing.T) {
			assert.False(t, CanTransition(unknown, status), "unknown -> %q must be forbidden", status)
			assert.False(t, CanTransition(status, unknown), "%q -> unknown must be forbidden", status)
		})
	}
}

func TestStatusTransitions_Complete(t *testing.T) {
	for _, status := range AllStatuses() {
		_, ok := statusTransitions[status]
		assert.True(t, ok, "status %q has no rule in statusTransitions", status)
	}
}

func TestAllowedFrom(t *testing.T) {
	tests := []struct {
		name string
		to   Status
		want []Status
	}{
		{name: "анкета заполнена", to: STATUS_PROFILE_FILLED, want: []Status{STATUS_NEW, STATUS_ON_KYC}},
		{name: "активен", to: STATUS_ACTIVE, want: []Status{STATUS_ON_KYC}},
		{name: "новый", to: STATUS_NEW, want: []Status{}},
		{name: "заблокирован", to: STATUS_BLOCKED, want: []Status{STATUS_NEW, STATUS_PROFILE_FILLED, STATUS_ON_KYC, STATUS_ACTIVE, STATUS_REJECTED}},
		{name: "статуса нет", to: STATUS_UNSPECIFIED, want: nil},
		{name: "неизвестный", to: Status("foo"), want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AllowedFrom(tc.to)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}

func TestAllowedFrom_ReturnsCopy(t *testing.T) {
	first := AllowedFrom(STATUS_PROFILE_FILLED)
	require.NotEmpty(t, first)
	first[0] = Status("foo")

	assert.Equal(t, []Status{STATUS_NEW, STATUS_ON_KYC}, AllowedFrom(STATUS_PROFILE_FILLED))
}

func TestAllowedFrom_ConsistentWithCanTransition(t *testing.T) {
	for _, from := range AllStatuses() {
		for _, to := range AllStatuses() {
			assert.Equal(t, CanTransition(from, to), slices.Contains(AllowedFrom(to), from),
				"AllowedFrom and CanTransition disagree on %q -> %q", from, to)
		}
	}
}

func TestAllowedTo(t *testing.T) {
	tests := []struct {
		name string
		from Status
		want []Status
	}{
		{name: "на kyc", from: STATUS_ON_KYC, want: []Status{STATUS_PROFILE_FILLED, STATUS_ACTIVE, STATUS_REJECTED, STATUS_BLOCKED}},
		{name: "новый", from: STATUS_NEW, want: []Status{STATUS_PROFILE_FILLED, STATUS_BLOCKED}},
		{name: "заблокирован", from: STATUS_BLOCKED, want: []Status{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.ElementsMatch(t, tc.want, AllowedTo(tc.from))
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	for _, status := range AllStatuses() {
		t.Run(string(status), func(t *testing.T) {
			assert.True(t, status.IsValid())
		})
	}

	t.Run("статуса нет", func(t *testing.T) {
		assert.False(t, STATUS_UNSPECIFIED.IsValid())
	})
	t.Run("неизвестный", func(t *testing.T) {
		assert.False(t, Status("foo").IsValid())
	})
}

func TestStatus_IsEditable(t *testing.T) {
	editable := map[Status]bool{STATUS_NEW: true, STATUS_PROFILE_FILLED: true}

	for _, status := range append(AllStatuses(), STATUS_UNSPECIFIED, Status("foo")) {
		t.Run(string(status), func(t *testing.T) {
			assert.Equal(t, editable[status], status.IsEditable())
		})
	}
}

func TestAllStatuses(t *testing.T) {
	want := []Status{STATUS_NEW, STATUS_PROFILE_FILLED, STATUS_ON_KYC, STATUS_ACTIVE, STATUS_REJECTED, STATUS_BLOCKED}

	got := AllStatuses()
	assert.ElementsMatch(t, want, got)
	assert.NotContains(t, got, STATUS_UNSPECIFIED)

	// Изменение результата не должно задевать следующий вызов.
	got[0] = Status("foo")
	assert.ElementsMatch(t, want, AllStatuses())
}
