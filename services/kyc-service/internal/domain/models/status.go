package models

import (
	"fmt"
	"slices"
)

// ApplicationStatus - статус заявки. Значение совпадает с меткой enum
// application_status в Postgres, поэтому пишется в нижнем регистре и уезжает в
// базу как есть.
type ApplicationStatus string

const (
	STATUS_UNSPECIFIED ApplicationStatus = ""

	STATUS_SUBMITTED      ApplicationStatus = "submitted"
	STATUS_AUTO_CHECKS    ApplicationStatus = "auto_checks"
	STATUS_AUTO_REJECTED  ApplicationStatus = "auto_rejected"
	STATUS_IN_REVIEW      ApplicationStatus = "in_review"
	STATUS_DOCS_REQUESTED ApplicationStatus = "docs_requested"
	STATUS_APPROVED       ApplicationStatus = "approved"
	STATUS_REJECTED       ApplicationStatus = "rejected"
)

var allStatuses = []ApplicationStatus{
	STATUS_SUBMITTED,
	STATUS_AUTO_CHECKS,
	STATUS_AUTO_REJECTED,
	STATUS_IN_REVIEW,
	STATUS_DOCS_REQUESTED,
	STATUS_APPROVED,
	STATUS_REJECTED,
}

// activeStatuses - заявка ещё в работе и занимает у клиента единственный слот
// активной заявки.
//
// Этот список обязан совпадать с условием частичного уникального индекса
// applications_active_user_id_key в миграции 000001: индекс - единственное, что
// на самом деле не даёт завести вторую заявку, а домен только объясняет отказ.
var activeStatuses = []ApplicationStatus{
	STATUS_SUBMITTED,
	STATUS_AUTO_CHECKS,
	STATUS_IN_REVIEW,
	STATUS_DOCS_REQUESTED,
}

type statusRule struct {
	From []ApplicationStatus
}

// statusTransitions - таблица переходов заявки:
//
//	submitted -> auto_checks -> in_review -> docs_requested -> submitted
//	                 |              |-> rejected
//	                 |              |-> approved
//	                 |-> auto_rejected
var statusTransitions = map[ApplicationStatus]statusRule{

	STATUS_SUBMITTED:   {From: []ApplicationStatus{STATUS_DOCS_REQUESTED}},
	STATUS_AUTO_CHECKS: {From: []ApplicationStatus{STATUS_SUBMITTED}},

	STATUS_AUTO_REJECTED: {From: []ApplicationStatus{STATUS_AUTO_CHECKS}},
	STATUS_IN_REVIEW:     {From: []ApplicationStatus{STATUS_AUTO_CHECKS}},

	STATUS_DOCS_REQUESTED: {From: []ApplicationStatus{STATUS_IN_REVIEW}},
	STATUS_APPROVED:       {From: []ApplicationStatus{STATUS_IN_REVIEW}},
	STATUS_REJECTED:       {From: []ApplicationStatus{STATUS_IN_REVIEW}},
}

func (s ApplicationStatus) IsValid() bool {
	return slices.Contains(allStatuses, s)
}

func (s ApplicationStatus) IsActive() bool {
	return slices.Contains(activeStatuses, s)
}

// CanTransition проверяет переход по таблице.
func CanTransition(from, to ApplicationStatus) bool {
	if !from.IsValid() || !to.IsValid() || from == to {
		return false
	}

	rule, ok := statusTransitions[to]
	if !ok {
		return false
	}

	return slices.Contains(rule.From, from)
}

// AllowedFrom отдаёт статусы, из которых можно перейти в to.
func AllowedFrom(to ApplicationStatus) []ApplicationStatus {
	rule, ok := statusTransitions[to]
	if !ok {
		return nil
	}

	return slices.Clone(rule.From)
}

// AllowedTo отдаёт статусы, в которые можно уйти из from. Нужен для внятного
// текста ошибки: "из in_review доступны docs_requested, approved, rejected".
func AllowedTo(from ApplicationStatus) []ApplicationStatus {
	targets := make([]ApplicationStatus, 0, len(allStatuses))

	for _, to := range allStatuses {
		if CanTransition(from, to) {
			targets = append(targets, to)
		}
	}

	return targets
}

// ValidateStatusTransitions проверяет, что таблица переходов не разошлась со
// списком статусов. Зовётся из main на старте.
func ValidateStatusTransitions() error {
	for _, status := range allStatuses {
		if _, ok := statusTransitions[status]; !ok {
			return fmt.Errorf("статус %q не описан в таблице переходов", status)
		}
	}

	for status, rule := range statusTransitions {
		if !status.IsValid() {
			return fmt.Errorf("таблица переходов содержит неизвестный статус %q", status)
		}

		for _, from := range rule.From {
			if !from.IsValid() {
				return fmt.Errorf("статус %q: неизвестный источник перехода %q", status, from)
			}

			if from == status {
				return fmt.Errorf("статус %q указан источником самого себя", status)
			}
		}
	}

	for _, status := range activeStatuses {
		if !status.IsValid() {
			return fmt.Errorf("список активных статусов содержит неизвестный статус %q", status)
		}
	}

	for _, status := range allStatuses {
		if hasExit := len(AllowedTo(status)) > 0; hasExit != status.IsActive() {
			return fmt.Errorf(
				"статус %q: активность (%v) разошлась с наличием переходов (%v)",
				status, status.IsActive(), hasExit,
			)
		}
	}

	return nil
}
