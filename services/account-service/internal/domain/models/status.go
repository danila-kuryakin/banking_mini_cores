package models

import "slices"

type AccountStatus string

const (
	STATUS_ACTIVE  AccountStatus = "active"
	STATUS_BLOCKED AccountStatus = "blocked"
	STATUS_CLOSED  AccountStatus = "closed"
)

var allStatuses = []AccountStatus{
	STATUS_ACTIVE,
	STATUS_BLOCKED,
	STATUS_CLOSED,
}

type statusRule struct {
	From []AccountStatus
}

// statusTransitions - правило на стороне целевого статуса: active <-> blocked, active -> closed.
var statusTransitions = map[AccountStatus]statusRule{
	STATUS_ACTIVE:  {From: []AccountStatus{STATUS_BLOCKED}},
	STATUS_BLOCKED: {From: []AccountStatus{STATUS_ACTIVE}},
	STATUS_CLOSED:  {From: []AccountStatus{STATUS_ACTIVE}},
}

func (s AccountStatus) IsValid() bool {
	return slices.Contains(allStatuses, s)
}

func CanTransition(from, to AccountStatus) bool {
	if !from.IsValid() || !to.IsValid() || from == to {
		return false
	}

	rule, ok := statusTransitions[to]
	if !ok {
		return false
	}

	return slices.Contains(rule.From, from)
}

func AllowedFrom(to AccountStatus) []AccountStatus {
	rule, ok := statusTransitions[to]
	if !ok {
		return nil
	}

	return slices.Clone(rule.From)
}
