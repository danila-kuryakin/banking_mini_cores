package models

import (
	"time"

	"github.com/google/uuid"
)

// RuleParams - пороги правила, лежат в af_rules.params как JSON.
type RuleParams struct {
	// Limits - суммы по валютам в минорных единицах; валюта без порога правилом не проверяется.
	Limits        map[Currency]int64 `json:"limits,omitempty"`
	MaxCount      int                `json:"max_count,omitempty"`
	WindowSeconds int                `json:"window_seconds,omitempty"`
}

// RuleConfig - строка af_rules.
type RuleConfig struct {
	Code        string
	Description string
	Enabled     bool
	Params      RuleParams
	UpdatedBy   *uuid.UUID
	UpdatedAt   time.Time
}

// RuleUpdate - правка правила офицером; nil-поле не меняется.
type RuleUpdate struct {
	Code      string
	Enabled   *bool
	Params    *RuleParams
	OfficerID uuid.UUID
}
