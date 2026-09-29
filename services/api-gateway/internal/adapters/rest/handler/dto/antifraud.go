package dto

import "time"

// RuleParams - пороги правила; каждое правило читает только свои поля.
type RuleParams struct {
	// Суммы по валютам в минорных единицах; валюта без порога правилом не проверяется.
	Limits        map[string]int64 `json:"limits,omitempty"`
	MaxCount      int32            `json:"max_count,omitempty" example:"5"`
	WindowSeconds int32            `json:"window_seconds,omitempty" example:"60"`
}

type Rule struct {
	Code        string     `json:"code" example:"velocity" enums:"single_amount_limit,velocity,daily_amount_limit,same_account"`
	Description string     `json:"description"`
	Enabled     bool       `json:"enabled"`
	Params      RuleParams `json:"params"`
	UpdatedBy   string     `json:"updated_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UpdateRule - не переданное поле не меняется, переданные params заменяют пороги целиком.
type UpdateRule struct {
	Enabled *bool       `json:"enabled,omitempty" example:"true"`
	Params  *RuleParams `json:"params,omitempty"`
}

type ListRules struct {
	Rules []Rule `json:"rules"`
}
