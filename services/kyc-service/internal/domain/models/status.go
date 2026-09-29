package models

import (
	"slices"
)

// ApplicationStatus - статус заявки. Значение совпадает с меткой enum
// application_status в Postgres, поэтому пишется в нижнем регистре и уезжает в
// базу как есть.
type ApplicationStatus string

const (
	//STATUS_UNSPECIFIED ApplicationStatus = ""

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

// AllowedFrom отдаёт статусы, из которых можно перейти в to.
func AllowedFrom(to ApplicationStatus) []ApplicationStatus {
	rule, ok := statusTransitions[to]
	if !ok {
		return nil
	}

	return slices.Clone(rule.From)
}
