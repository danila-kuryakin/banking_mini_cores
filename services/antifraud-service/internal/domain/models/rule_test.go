package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleParams_Marshal(t *testing.T) {
	tests := []struct {
		name   string
		params RuleParams
		want   string
	}{
		{name: "zero value", params: RuleParams{}, want: `{}`},
		{name: "limits only", params: RuleParams{Limits: map[Currency]int64{"RUB": 1000}}, want: `{"limits":{"RUB":1000}}`},
		{name: "velocity params", params: RuleParams{MaxCount: 5, WindowSeconds: 60}, want: `{"max_count":5,"window_seconds":60}`},
		{name: "empty limits map omitted", params: RuleParams{Limits: map[Currency]int64{}}, want: `{}`},
		{
			name:   "all fields",
			params: RuleParams{Limits: map[Currency]int64{"RUB": 1000, "USD": 50}, MaxCount: 3, WindowSeconds: 3600},
			want:   `{"limits":{"RUB":1000,"USD":50},"max_count":3,"window_seconds":3600}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.params)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(raw))
		})
	}
}

func TestRuleParams_Unmarshal(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want RuleParams
	}{
		{
			name: "all keys",
			raw:  `{"limits":{"RUB":1000},"max_count":5,"window_seconds":60}`,
			want: RuleParams{Limits: map[Currency]int64{"RUB": 1000}, MaxCount: 5, WindowSeconds: 60},
		},
		{name: "empty object", raw: `{}`, want: RuleParams{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got RuleParams
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &got))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRuleParams_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		params RuleParams
	}{
		{name: "limits only", params: RuleParams{Limits: map[Currency]int64{"RUB": 1000}}},
		{name: "velocity params", params: RuleParams{MaxCount: 5, WindowSeconds: 60}},
		{name: "all fields", params: RuleParams{Limits: map[Currency]int64{"RUB": 1000, "USD": 50}, MaxCount: 3, WindowSeconds: 3600}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.params)
			require.NoError(t, err)
			var got RuleParams
			require.NoError(t, json.Unmarshal(raw, &got))
			assert.Equal(t, tt.params, got)
		})
	}
}
