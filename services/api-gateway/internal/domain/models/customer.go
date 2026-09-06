package models

type Status string

const (
	STATUS_NEW            Status = "new"
	STATUS_PROFILE_FILLED Status = "profile_filled"
	STATUS_ON_KYC         Status = "on_kyc"
	STATUS_ACTIVE         Status = "active"
	STATUS_REJECTED       Status = "rejected"
	STATUS_BLOCKED        Status = "blocked"
)
