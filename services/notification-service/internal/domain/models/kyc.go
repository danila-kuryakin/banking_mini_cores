package models

// События kyc-service из топика kyc.events.
const (
	EVENT_APPLICATION_SUBMITTED = "kyc.application_submitted"
	EVENT_APPLICATION_APPROVED  = "kyc.application_approved"
	EVENT_APPLICATION_REJECTED  = "kyc.application_rejected"
	EVENT_DOCS_REQUESTED        = "kyc.docs_requested"

	APPLICATION_STATUS_REJECTED      = "rejected"
	APPLICATION_STATUS_AUTO_REJECTED = "auto_rejected"
)

// ApplicationPayload - payload событий заявки.
type ApplicationPayload struct {
	ApplicationID  string   `json:"application_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Reason         string   `json:"reason"`
	RequestedTypes []string `json:"requested_types"`
}
