package grpc

import (
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"google.golang.org/genproto/googleapis/type/date"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

func profileToProto(profile models.Profile) *customerv1.Profile {
	return &customerv1.Profile{
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		BirthDate:   dateToProto(profile.BirthDate),
		Citizenship: profile.Citizenship,
		Phone:       profile.Phone,
	}
}

func statusToProto(status models.Status) customerv1.CustomerStatus {
	switch status {
	case models.STATUS_NEW:
		return customerv1.CustomerStatus_NEW
	case models.STATUS_PROFILE_FILLED:
		return customerv1.CustomerStatus_PROFILE_FILLED
	case models.STATUS_ON_KYC:
		return customerv1.CustomerStatus_ON_KYC
	case models.STATUS_ACTIVE:
		return customerv1.CustomerStatus_ACTIVE
	case models.STATUS_REJECTED:
		return customerv1.CustomerStatus_REJECTED
	case models.STATUS_BLOCKED:
		return customerv1.CustomerStatus_BLOCKED
	default:
		return customerv1.CustomerStatus_NEW
	}
}

func dateToProto(in *time.Time) *date.Date {
	if in == nil {
		return nil
	}

	return &date.Date{
		Year:  int32(in.Year()),
		Month: int32(in.Month()),
		Day:   int32(in.Day()),
	}
}

func dateFromProto(in *date.Date) *time.Time {
	if in == nil || in.Year == 0 || in.Month == 0 || in.Day == 0 {
		return nil
	}

	parsed := time.Date(int(in.Year), time.Month(in.Month), int(in.Day), 0, 0, 0, 0, time.UTC)

	return &parsed
}
