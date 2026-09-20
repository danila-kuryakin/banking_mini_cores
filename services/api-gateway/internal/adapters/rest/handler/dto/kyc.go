package dto

import (
	"time"
)

// Application - заявка на верификацию. Статусы и типы документов отдаются
// доменными именами в нижнем регистре, как и статус клиента.
type Application struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Status         string     `json:"status" example:"in_review" enums:"submitted,auto_checks,auto_rejected,in_review,docs_requested,approved,rejected"`
	ReviewerID     string     `json:"reviewer_id,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	RequestedTypes []string   `json:"requested_types,omitempty" example:"proof_of_address"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	DecidedAt      *time.Time `json:"decided_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Checks         []Check    `json:"checks,omitempty"`
	Events         []Event    `json:"events,omitempty"`
}

// Check - результат авто-проверки.
type Check struct {
	ID        string    `json:"id"`
	Type      string    `json:"type" example:"sanctions" enums:"sanctions,age"`
	Result    string    `json:"result" example:"passed" enums:"passed,failed"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

// Event - запись аудита заявки. Пустой actor_id означает системный переход.
type Event struct {
	ID         string    `json:"id"`
	FromStatus string    `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	ActorID    string    `json:"actor_id,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ListApplicationsQuery struct {
	Statuses   []string `form:"status" example:"in_review"`
	Reviewer   string   `form:"reviewer_id"`
	Unassigned bool     `form:"unassigned"`
	Limit      int32    `form:"limit"`
	Offset     int32    `form:"offset"`
}

type ListApplications struct {
	Applications []Application `json:"applications"`
}

type RejectApplication struct {
	Reason string `json:"reason" example:"документы не совпадают с анкетой"`
}

type ApproveApplication struct {
	Comment string `json:"comment" example:"всё сходится"`
}

type RequestDocuments struct {
	Types   []string `json:"types" example:"proof_of_address" enums:"passport,selfie,proof_of_address"`
	Comment string   `json:"comment" example:"нужна справка о месте жительства"`
}

type AddSanctionEntry struct {
	FirstName string `json:"first_name" example:"Ivan"`
	LastName  string `json:"last_name" example:"Petrov"`
	BirthDate string `json:"birth_date" example:"1980-01-15"`
	Source    string `json:"source" example:"OFAC SDN"`
}

type SanctionEntry struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	BirthDate string `json:"birth_date" example:"1980-01-15"`
	Source    string `json:"source"`
}

type ListSanctionsQuery struct {
	Limit  int32 `form:"limit"`
	Offset int32 `form:"offset"`
}

type ListSanctions struct {
	Entries []SanctionEntry `json:"entries"`
}
