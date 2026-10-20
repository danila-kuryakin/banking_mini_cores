package models

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testAllStatuses - все 7 статусов заявки из спецификации.
func testAllStatuses() []ApplicationStatus {
	return []ApplicationStatus{
		STATUS_SUBMITTED,
		STATUS_AUTO_CHECKS,
		STATUS_AUTO_REJECTED,
		STATUS_IN_REVIEW,
		STATUS_DOCS_REQUESTED,
		STATUS_APPROVED,
		STATUS_REJECTED,
	}
}

// testExpectedAllowedFrom - ожидаемые источники для каждого целевого статуса (R1).
func testExpectedAllowedFrom() map[ApplicationStatus][]ApplicationStatus {
	return map[ApplicationStatus][]ApplicationStatus{
		STATUS_SUBMITTED:      {STATUS_DOCS_REQUESTED},
		STATUS_AUTO_CHECKS:    {STATUS_SUBMITTED},
		STATUS_AUTO_REJECTED:  {STATUS_AUTO_CHECKS},
		STATUS_IN_REVIEW:      {STATUS_AUTO_CHECKS},
		STATUS_DOCS_REQUESTED: {STATUS_IN_REVIEW},
		STATUS_APPROVED:       {STATUS_IN_REVIEW},
		STATUS_REJECTED:       {STATUS_IN_REVIEW},
	}
}

func TestAllowedFrom_TransitionMatrix(t *testing.T) {
	type transition struct {
		from ApplicationStatus
		to   ApplicationStatus
	}
	// ровно 7 разрешённых пар из ТЗ
	allowedPairs := map[transition]bool{
		{STATUS_DOCS_REQUESTED, STATUS_SUBMITTED}:  true,
		{STATUS_SUBMITTED, STATUS_AUTO_CHECKS}:     true,
		{STATUS_AUTO_CHECKS, STATUS_AUTO_REJECTED}: true,
		{STATUS_AUTO_CHECKS, STATUS_IN_REVIEW}:     true,
		{STATUS_IN_REVIEW, STATUS_DOCS_REQUESTED}:  true,
		{STATUS_IN_REVIEW, STATUS_APPROVED}:        true,
		{STATUS_IN_REVIEW, STATUS_REJECTED}:        true,
	}

	type testCase struct {
		name    string
		from    ApplicationStatus
		to      ApplicationStatus
		allowed bool
	}
	// таблица всех 49 пар 7x7, переходы в себя получают префикс self_
	var tests []testCase
	for _, from := range testAllStatuses() {
		for _, to := range testAllStatuses() {
			name := string(from) + "->" + string(to)
			if from == to {
				name = "self_" + string(from)
			}
			tests = append(tests, testCase{
				name:    name,
				from:    from,
				to:      to,
				allowed: allowedPairs[transition{from, to}],
			})
		}
	}
	assert.Len(t, tests, 49)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Contains(AllowedFrom(tt.to), tt.from)
			assert.Equal(t, tt.allowed, got)
		})
	}
}

func TestAllowedFrom_Exact(t *testing.T) {
	for to, want := range testExpectedAllowedFrom() {
		t.Run(string(to), func(t *testing.T) {
			assert.ElementsMatch(t, want, AllowedFrom(to))
		})
	}
}

func TestAllowedFrom_UnknownStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ApplicationStatus
	}{
		{name: "empty", status: ""},
		{name: "draft", status: "draft"},
		{name: "upper_case", status: "SUBMITTED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Empty(t, AllowedFrom(tt.status))
		})
	}
}

func TestAllowedFrom_ReturnsCopy(t *testing.T) {
	for to, want := range testExpectedAllowedFrom() {
		t.Run(string(to), func(t *testing.T) {
			first := AllowedFrom(to)
			for i := range first {
				first[i] = "draft"
			}
			assert.ElementsMatch(t, want, AllowedFrom(to))
		})
	}
}

func TestApplicationStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status ApplicationStatus
		want   bool
	}{
		{name: "submitted", status: STATUS_SUBMITTED, want: true},
		{name: "auto_checks", status: STATUS_AUTO_CHECKS, want: true},
		{name: "auto_rejected", status: STATUS_AUTO_REJECTED, want: true},
		{name: "in_review", status: STATUS_IN_REVIEW, want: true},
		{name: "docs_requested", status: STATUS_DOCS_REQUESTED, want: true},
		{name: "approved", status: STATUS_APPROVED, want: true},
		{name: "rejected", status: STATUS_REJECTED, want: true},
		{name: "empty", status: "", want: false},
		{name: "draft", status: "draft", want: false},
		{name: "upper_case", status: "SUBMITTED", want: false},
		{name: "leading_space", status: " submitted", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestApplicationStatus_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status ApplicationStatus
		want   bool
	}{
		{name: "submitted", status: STATUS_SUBMITTED, want: true},
		{name: "auto_checks", status: STATUS_AUTO_CHECKS, want: true},
		{name: "in_review", status: STATUS_IN_REVIEW, want: true},
		{name: "docs_requested", status: STATUS_DOCS_REQUESTED, want: true},
		{name: "auto_rejected", status: STATUS_AUTO_REJECTED, want: false},
		{name: "approved", status: STATUS_APPROVED, want: false},
		{name: "rejected", status: STATUS_REJECTED, want: false},
		{name: "empty", status: "", want: false},
		{name: "draft", status: "draft", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsActive())
		})
	}
}

func TestApplicationStatus_IsActiveMatchesHasExit(t *testing.T) {
	for _, status := range testAllStatuses() {
		t.Run(string(status), func(t *testing.T) {
			// у статуса есть выход, если он источник хотя бы для одного целевого статуса
			hasExit := false
			for _, to := range testAllStatuses() {
				hasExit = hasExit || slices.Contains(AllowedFrom(to), status)
			}
			assert.Equal(t, hasExit, status.IsActive())
		})
	}
}
