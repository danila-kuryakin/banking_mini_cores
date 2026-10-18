package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerdictConstants(t *testing.T) {
	assert.Equal(t, Verdict("allow"), VERDICT_ALLOW)
	assert.Equal(t, Verdict("deny"), VERDICT_DENY)
}

func TestVerdictFor(t *testing.T) {
	tests := []struct {
		name string
		hits []Hit
		want Verdict
	}{
		{name: "nil hits", hits: nil, want: VERDICT_ALLOW},
		{name: "empty hits", hits: []Hit{}, want: VERDICT_ALLOW},
		{name: "one hit", hits: []Hit{{RuleCode: "velocity", Reason: "3 transfers"}}, want: VERDICT_DENY},
		{name: "two hits", hits: []Hit{{RuleCode: "velocity", Reason: "3 transfers"}, {RuleCode: "same_account", Reason: "same"}}, want: VERDICT_DENY},
		{name: "empty hit counts", hits: []Hit{{}}, want: VERDICT_DENY},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, VerdictFor(tt.hits))
		})
	}
}

func TestCheck_Reasons(t *testing.T) {
	tests := []struct {
		name string
		hits []Hit
		want []string
	}{
		{
			name: "two hits",
			hits: []Hit{{RuleCode: "velocity", Reason: "3 transfers"}, {RuleCode: "same_account", Reason: "same"}},
			want: []string{"velocity: 3 transfers", "same_account: same"},
		},
		{
			name: "one hit",
			hits: []Hit{{RuleCode: "single_amount_limit", Reason: "amount 1000 exceeds 500"}},
			want: []string{"single_amount_limit: amount 1000 exceeds 500"},
		},
		{
			name: "empty reason",
			hits: []Hit{{RuleCode: "velocity", Reason: ""}},
			want: []string{"velocity: "},
		},
		{
			name: "order of hits kept",
			hits: []Hit{{RuleCode: "velocity", Reason: "c"}, {RuleCode: "same_account", Reason: "b"}, {RuleCode: "daily_amount_limit", Reason: "a"}},
			want: []string{"velocity: c", "same_account: b", "daily_amount_limit: a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := &Check{Hits: tt.hits}
			assert.Equal(t, tt.want, check.Reasons())
		})
	}
}

func TestCheck_ReasonsNoHits(t *testing.T) {
	tests := []struct {
		name string
		hits []Hit
	}{
		{name: "nil hits", hits: nil},
		{name: "empty hits", hits: []Hit{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := &Check{Hits: tt.hits}
			assert.Len(t, check.Reasons(), 0)
		})
	}
}
