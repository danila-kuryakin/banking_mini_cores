package models

import (
	"slices"
	"time"
)

type Status string

const (
	STATUS_UNSPECIFIED    Status = ""
	STATUS_NEW            Status = "new"
	STATUS_PROFILE_FILLED Status = "profile_filled"
	STATUS_ON_KYC         Status = "on_kyc"
	STATUS_ACTIVE         Status = "active"
	STATUS_REJECTED       Status = "rejected"
	STATUS_BLOCKED        Status = "blocked"
)

var allStatuses = []Status{
	STATUS_NEW,
	STATUS_PROFILE_FILLED,
	STATUS_ON_KYC,
	STATUS_ACTIVE,
	STATUS_REJECTED,
	STATUS_BLOCKED,
}

type statusRule struct {
	From    []Status
	FromAny bool
}

// statusTransitions - таблица переходов.
var statusTransitions = map[Status]statusRule{
	STATUS_NEW: {From: nil},

	STATUS_PROFILE_FILLED: {From: []Status{STATUS_NEW}},
	STATUS_ON_KYC:         {From: []Status{STATUS_PROFILE_FILLED}},
	STATUS_ACTIVE:         {From: []Status{STATUS_ON_KYC}},
	STATUS_REJECTED:       {From: []Status{STATUS_ON_KYC}},

	STATUS_BLOCKED: {FromAny: true},
}

// StatusChange - результат смены статуса. Previous совпадает с Current, если
// перехода не было: клиент уже находился в целевом статусе.
type StatusChange struct {
	Previous  Status
	Current   Status
	ChangedAt time.Time
}

// AllStatuses отдаёт копию перечня реальных статусов - без STATUS_UNSPECIFIED.
func AllStatuses() []Status {
	return slices.Clone(allStatuses)
}

// IsValid сообщает, известен ли статус домену. STATUS_UNSPECIFIED невалиден.
func (s Status) IsValid() bool {
	return slices.Contains(allStatuses, s)
}

// CanTransition проверяет переход по таблице.
func CanTransition(from, to Status) bool {
	if !from.IsValid() || !to.IsValid() || from == to {
		return false
	}

	rule, ok := statusTransitions[to]
	if !ok {
		return false
	}

	if rule.FromAny {
		return true
	}

	return slices.Contains(rule.From, from)
}

// AllowedFrom отдаёт статусы, из которых можно перейти в to.
func AllowedFrom(to Status) []Status {
	rule, ok := statusTransitions[to]
	if !ok {
		return nil
	}

	if !rule.FromAny {
		return slices.Clone(rule.From)
	}

	sources := make([]Status, 0, len(allStatuses))

	for _, from := range allStatuses {
		if from != to {
			sources = append(sources, from)
		}
	}

	return sources
}

// AllowedTo отдаёт статусы, в которые можно уйти из from. Нужен для внятного
// текста ошибки: "из on_kyc доступны active, rejected, blocked".
func AllowedTo(from Status) []Status {
	targets := make([]Status, 0, len(allStatuses))

	for _, to := range allStatuses {
		if CanTransition(from, to) {
			targets = append(targets, to)
		}
	}

	return targets
}
