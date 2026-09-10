package grpc

import (
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/types/known/timestamppb"

	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

func customerToProto(customer *models.Customer) *customerv1.Customer {
	return &customerv1.Customer{
		Id:              customer.ID.String(),
		UserId:          customer.UserID.String(),
		Status:          statusToProto(customer.Status),
		StatusChangedAt: timestamppb.New(customer.StatusChangedAt),
		Profile:         profileToProto(customer.Profile),
		CreatedAt:       timestamppb.New(customer.CreatedAt),
		UpdatedAt:       timestamppb.New(customer.UpdatedAt),
	}
}

func profileToProto(profile models.Profile) *customerv1.Profile {
	return &customerv1.Profile{
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		BirthDate:   dateToProto(profile.BirthDate),
		Citizenship: profile.Citizenship,
		Phone:       profile.Phone,
	}
}

// statusPairs - соответствие доменного статуса значению enum в контракте.
var statusPairs = []struct {
	domain models.Status
	proto  customerv1.CustomerStatus
}{
	{models.STATUS_NEW, customerv1.CustomerStatus_NEW},
	{models.STATUS_PROFILE_FILLED, customerv1.CustomerStatus_PROFILE_FILLED},
	{models.STATUS_ON_KYC, customerv1.CustomerStatus_ON_KYC},
	{models.STATUS_ACTIVE, customerv1.CustomerStatus_ACTIVE},
	{models.STATUS_REJECTED, customerv1.CustomerStatus_REJECTED},
	{models.STATUS_BLOCKED, customerv1.CustomerStatus_BLOCKED},
}

var statusToProtoMap, statusFromProtoMap = buildStatusMaps()

func buildStatusMaps() (map[models.Status]customerv1.CustomerStatus, map[customerv1.CustomerStatus]models.Status) {
	toProto := make(map[models.Status]customerv1.CustomerStatus, len(statusPairs))
	fromProto := make(map[customerv1.CustomerStatus]models.Status, len(statusPairs))

	for _, pair := range statusPairs {
		toProto[pair.domain] = pair.proto
		fromProto[pair.proto] = pair.domain
	}

	return toProto, fromProto
}

// statusToProto переводит статус в контракт.
func statusToProto(status models.Status) customerv1.CustomerStatus {
	return statusToProtoMap[status]
}

// statusFromProto переводит статус из контракта.
func statusFromProto(status customerv1.CustomerStatus) models.Status {
	return statusFromProtoMap[status]
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
