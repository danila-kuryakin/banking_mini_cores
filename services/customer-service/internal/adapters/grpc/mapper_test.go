package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

func TestStatusPairs_CoverAllStatuses(t *testing.T) {
	seen := make(map[models.Status]int, len(statusPairs))
	for _, pair := range statusPairs {
		seen[pair.domain]++
	}

	for _, s := range models.AllStatuses() {
		assert.Equal(t, 1, seen[s], "status %q must appear in statusPairs exactly once", s)
	}
	assert.Len(t, statusPairs, len(models.AllStatuses()))
}

func TestStatus_RoundTrip(t *testing.T) {
	for _, s := range models.AllStatuses() {
		t.Run(string(s), func(t *testing.T) {
			assert.Equal(t, s, statusFromProto(statusToProto(s)))
		})
	}
}

func TestStatusToProto_Unknown(t *testing.T) {
	assert.Equal(t, customerv1.CustomerStatus_UNSPECIFIED, statusToProto(models.Status("foo")))
	assert.Equal(t, customerv1.CustomerStatus_UNSPECIFIED, statusToProto(models.STATUS_UNSPECIFIED))
}

func TestStatusFromProto_Unknown(t *testing.T) {
	assert.Equal(t, models.STATUS_UNSPECIFIED, statusFromProto(customerv1.CustomerStatus_UNSPECIFIED))
	assert.Equal(t, models.STATUS_UNSPECIFIED, statusFromProto(customerv1.CustomerStatus(99)))
}

func TestDateToProto(t *testing.T) {
	assert.Nil(t, dateToProto(nil))

	leapDay := time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, &date.Date{Year: 2000, Month: 2, Day: 29}, dateToProto(&leapDay))
}

func TestDateFromProto(t *testing.T) {
	leapDay := time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		in      *date.Date
		want    *time.Time
		wantErr bool
	}{
		{name: "nil", in: nil},
		{name: "все нули", in: &date.Date{}},
		{name: "нет месяца", in: &date.Date{Year: 2000, Month: 0, Day: 1}, wantErr: true},
		{name: "нет года", in: &date.Date{Year: 0, Month: 1, Day: 1}, wantErr: true},
		{name: "нет дня", in: &date.Date{Year: 2000, Month: 1, Day: 0}, wantErr: true},
		{name: "29 февраля не високосного", in: &date.Date{Year: 2001, Month: 2, Day: 29}, wantErr: true},
		{name: "31 апреля", in: &date.Date{Year: 2000, Month: 4, Day: 31}, wantErr: true},
		{name: "13-й месяц", in: &date.Date{Year: 2000, Month: 13, Day: 1}, wantErr: true},
		{name: "29 февраля високосного", in: &date.Date{Year: 2000, Month: 2, Day: 29}, want: &leapDay},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := dateFromProto(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDate_RoundTrip(t *testing.T) {
	birthDate := time.Date(1990, time.May, 17, 0, 0, 0, 0, time.UTC)

	got, err := dateFromProto(dateToProto(&birthDate))

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, birthDate.Equal(*got), "round trip changed the date: %v != %v", birthDate, *got)
}

func TestCustomerToProto(t *testing.T) {
	customer := testCustomer(models.STATUS_PROFILE_FILLED)

	got := customerToProto(customer)

	assert.Equal(t, TEST_CUSTOMER_ID, got.Id)
	assert.Equal(t, TEST_USER_ID, got.UserId)
	assert.Equal(t, customerv1.CustomerStatus_PROFILE_FILLED, got.Status)
	assert.Equal(t, testTime, got.StatusChangedAt.AsTime())
	assert.Equal(t, testTime, got.CreatedAt.AsTime())
	assert.Equal(t, testTime, got.UpdatedAt.AsTime())

	require.NotNil(t, got.Profile)
	assert.Equal(t, customer.Profile.FirstName, got.Profile.FirstName)
	assert.Equal(t, customer.Profile.LastName, got.Profile.LastName)
	assert.Equal(t, customer.Profile.Citizenship, got.Profile.Citizenship)
	assert.Equal(t, customer.Profile.Phone, got.Profile.Phone)
	assert.Equal(t, dateToProto(customer.Profile.BirthDate), got.Profile.BirthDate)
}

func TestCustomerToProto_NoBirthDate(t *testing.T) {
	customer := testCustomer(models.STATUS_NEW)
	customer.Profile.BirthDate = nil

	got := customerToProto(customer)

	require.NotNil(t, got.Profile)
	assert.Nil(t, got.Profile.BirthDate)
}
