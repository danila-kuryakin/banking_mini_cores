package models

import "time"

// CustomerStatus - статус клиента в customer-service. Копия чужого словаря:
// kyc его читает и двигает, поэтому значения обязаны совпадать с метками enum
// customer_status в той базе.
type CustomerStatus string

const (
	CUSTOMER_STATUS_UNSPECIFIED    CustomerStatus = ""
	CUSTOMER_STATUS_NEW            CustomerStatus = "new"
	CUSTOMER_STATUS_PROFILE_FILLED CustomerStatus = "profile_filled"
	CUSTOMER_STATUS_ON_KYC         CustomerStatus = "on_kyc"
	CUSTOMER_STATUS_ACTIVE         CustomerStatus = "active"
	CUSTOMER_STATUS_REJECTED       CustomerStatus = "rejected"
	CUSTOMER_STATUS_BLOCKED        CustomerStatus = "blocked"
)

// CustomerProfile - то, что kyc берёт из анкеты: имя и дата рождения для
// авто-проверок плюс текущий статус клиента.
type CustomerProfile struct {
	FirstName string
	LastName  string
	BirthDate *time.Time
	Status    CustomerStatus
}

// IsComplete повторяет условие полноты анкеты из customer-service: без этих
// полей авто-проверкам нечего сверять.
func (p CustomerProfile) IsComplete() bool {
	return p.FirstName != "" && p.LastName != "" && p.BirthDate != nil
}
