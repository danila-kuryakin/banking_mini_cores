package client

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/customer/v1"
)

var (
	TEST_USER_ID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_ACTOR_ID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

const TEST_REASON = "kyc application submitted"

// timePtr возвращает указатель на дату в UTC
func timePtr(year int, month time.Month, day int) *time.Time {
	birthDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &birthDate
}

// actorPtr возвращает указатель на копию идентификатора офицера
func actorPtr(actorID uuid.UUID) *uuid.UUID {
	return &actorID
}

func TestCustomerClient_GetProfile(t *testing.T) {
	tests := []struct {
		name     string
		customer *customerv1.Customer
		want     *models.CustomerProfile
	}{
		{
			name: "full profile from spec example",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_PROFILE_FILLED,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1990, Month: 5, Day: 15}},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: timePtr(1990, time.May, 15), Status: models.CUSTOMER_STATUS_PROFILE_FILLED},
		},
		{
			name:     "nil profile keeps status",
			customer: &customerv1.Customer{Status: customerv1.CustomerStatus_ACTIVE},
			want:     &models.CustomerProfile{FirstName: "", LastName: "", BirthDate: nil, Status: models.CUSTOMER_STATUS_ACTIVE},
		},
		{
			name: "nil birth date",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_NEW,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov"},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: nil, Status: models.CUSTOMER_STATUS_NEW},
		},
		{
			name: "zero year gives nil birth date",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_PROFILE_FILLED,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 0, Month: 5, Day: 15}},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: nil, Status: models.CUSTOMER_STATUS_PROFILE_FILLED},
		},
		{
			name: "zero month gives nil birth date",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_PROFILE_FILLED,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1990, Month: 0, Day: 15}},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: nil, Status: models.CUSTOMER_STATUS_PROFILE_FILLED},
		},
		{
			name: "zero day gives nil birth date",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_PROFILE_FILLED,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1990, Month: 5, Day: 0}},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: nil, Status: models.CUSTOMER_STATUS_PROFILE_FILLED},
		},
		{
			name: "leap day birth date",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_PROFILE_FILLED,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 2000, Month: 2, Day: 29}},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: timePtr(2000, time.February, 29), Status: models.CUSTOMER_STATUS_PROFILE_FILLED},
		},
		{
			name: "minimal non-zero birth date",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_PROFILE_FILLED,
				Profile: &customerv1.Profile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1, Month: 1, Day: 1}},
			},
			want: &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: timePtr(1, time.January, 1), Status: models.CUSTOMER_STATUS_PROFILE_FILLED},
		},
		{
			name: "names passed as is without trimming",
			customer: &customerv1.Customer{
				Status:  customerv1.CustomerStatus_NEW,
				Profile: &customerv1.Profile{FirstName: "  Ivan ", LastName: ""},
			},
			want: &models.CustomerProfile{FirstName: "  Ivan ", LastName: "", BirthDate: nil, Status: models.CUSTOMER_STATUS_NEW},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customers := &fakeCustomerServiceClient{getCustomerResp: tt.customer}
			customerClient := NewCustomerClient(customers, &fakeCustomerStatusServiceClient{})

			got, err := customerClient.GetProfile(t.Context(), TEST_USER_ID)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assertProtoCalls(t, []*customerv1.GetCustomerRequest{{UserId: TEST_USER_ID.String()}}, customers.getCustomerCalls)
		})
	}
}

func TestCustomerClient_GetProfile_NotFound(t *testing.T) {
	customers := &fakeCustomerServiceClient{getCustomerErr: status.Error(codes.NotFound, "customer not found")}
	customerClient := NewCustomerClient(customers, &fakeCustomerStatusServiceClient{})

	got, err := customerClient.GetProfile(t.Context(), TEST_USER_ID)

	require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	assert.Nil(t, got)
}

func TestCustomerClient_GetProfile_UpstreamErrors(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		wantCode codes.Code
	}{
		{name: "unavailable", upstream: status.Error(codes.Unavailable, "connection refused"), wantCode: codes.Unavailable},
		{name: "internal", upstream: status.Error(codes.Internal, "db is down"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customers := &fakeCustomerServiceClient{getCustomerErr: tt.upstream}
			customerClient := NewCustomerClient(customers, &fakeCustomerStatusServiceClient{})

			got, err := customerClient.GetProfile(t.Context(), TEST_USER_ID)

			require.ErrorIs(t, err, tt.upstream)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.False(t, errors.Is(err, domain.ErrCustomerNotFound))
			assert.Nil(t, got)
		})
	}
}

// Инвариант: каждое proto-значение статуса переводится в свой доменный статус
func TestCustomerClient_GetProfile_StatusMapping(t *testing.T) {
	tests := []struct {
		name  string
		proto customerv1.CustomerStatus
		want  models.CustomerStatus
	}{
		{name: "new", proto: customerv1.CustomerStatus_NEW, want: models.CUSTOMER_STATUS_NEW},
		{name: "profile_filled", proto: customerv1.CustomerStatus_PROFILE_FILLED, want: models.CUSTOMER_STATUS_PROFILE_FILLED},
		{name: "on_kyc", proto: customerv1.CustomerStatus_ON_KYC, want: models.CUSTOMER_STATUS_ON_KYC},
		{name: "active", proto: customerv1.CustomerStatus_ACTIVE, want: models.CUSTOMER_STATUS_ACTIVE},
		{name: "rejected", proto: customerv1.CustomerStatus_REJECTED, want: models.CUSTOMER_STATUS_REJECTED},
		{name: "blocked", proto: customerv1.CustomerStatus_BLOCKED, want: models.CUSTOMER_STATUS_BLOCKED},
		{name: "unspecified", proto: customerv1.CustomerStatus_UNSPECIFIED, want: models.CUSTOMER_STATUS_UNSPECIFIED},
		{name: "unknown value", proto: customerv1.CustomerStatus(99), want: models.CUSTOMER_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customers := &fakeCustomerServiceClient{getCustomerResp: &customerv1.Customer{Status: tt.proto}}
			customerClient := NewCustomerClient(customers, &fakeCustomerStatusServiceClient{})

			got, err := customerClient.GetProfile(t.Context(), TEST_USER_ID)

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.Status)
		})
	}
}

func TestCustomerClient_SetStatus(t *testing.T) {
	tests := []struct {
		name    string
		to      models.CustomerStatus
		reason  string
		actorID *uuid.UUID
		want    *customerv1.SetStatusRequest
	}{
		{
			name:   "system transition from spec example",
			to:     models.CUSTOMER_STATUS_ON_KYC,
			reason: TEST_REASON,
			want:   &customerv1.SetStatusRequest{UserId: TEST_USER_ID.String(), Status: customerv1.CustomerStatus_ON_KYC, Reason: TEST_REASON, ActorId: ""},
		},
		{
			name:    "officer transition with actor",
			to:      models.CUSTOMER_STATUS_REJECTED,
			reason:  "fraud",
			actorID: actorPtr(TEST_ACTOR_ID),
			want:    &customerv1.SetStatusRequest{UserId: TEST_USER_ID.String(), Status: customerv1.CustomerStatus_REJECTED, Reason: "fraud", ActorId: TEST_ACTOR_ID.String()},
		},
		{
			name: "empty reason",
			to:   models.CUSTOMER_STATUS_ACTIVE,
			want: &customerv1.SetStatusRequest{UserId: TEST_USER_ID.String(), Status: customerv1.CustomerStatus_ACTIVE, Reason: "", ActorId: ""},
		},
		{
			name:   "unspecified status",
			to:     models.CUSTOMER_STATUS_UNSPECIFIED,
			reason: TEST_REASON,
			want:   &customerv1.SetStatusRequest{UserId: TEST_USER_ID.String(), Status: customerv1.CustomerStatus_UNSPECIFIED, Reason: TEST_REASON, ActorId: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statuses := &fakeCustomerStatusServiceClient{}
			customerClient := NewCustomerClient(&fakeCustomerServiceClient{}, statuses)

			err := customerClient.SetStatus(t.Context(), TEST_USER_ID, tt.to, tt.reason, tt.actorID)

			require.NoError(t, err)
			assertProtoCalls(t, []*customerv1.SetStatusRequest{tt.want}, statuses.setStatusCalls)
		})
	}
}

func TestCustomerClient_SetStatus_NotFound(t *testing.T) {
	statuses := &fakeCustomerStatusServiceClient{setStatusErr: status.Error(codes.NotFound, "customer not found")}
	customerClient := NewCustomerClient(&fakeCustomerServiceClient{}, statuses)

	err := customerClient.SetStatus(t.Context(), TEST_USER_ID, models.CUSTOMER_STATUS_ON_KYC, TEST_REASON, nil)

	require.ErrorIs(t, err, domain.ErrCustomerNotFound)
}

func TestCustomerClient_SetStatus_UpstreamErrors(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		wantCode codes.Code
	}{
		{name: "unavailable", upstream: status.Error(codes.Unavailable, "connection refused"), wantCode: codes.Unavailable},
		{name: "internal", upstream: status.Error(codes.Internal, "db is down"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statuses := &fakeCustomerStatusServiceClient{setStatusErr: tt.upstream}
			customerClient := NewCustomerClient(&fakeCustomerServiceClient{}, statuses)

			err := customerClient.SetStatus(t.Context(), TEST_USER_ID, models.CUSTOMER_STATUS_ON_KYC, TEST_REASON, nil)

			require.ErrorIs(t, err, tt.upstream)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.False(t, errors.Is(err, domain.ErrCustomerNotFound))
		})
	}
}

// Инвариант: каждый доменный статус уходит в запрос своим proto-значением
func TestCustomerClient_SetStatus_StatusMapping(t *testing.T) {
	tests := []struct {
		name string
		to   models.CustomerStatus
		want customerv1.CustomerStatus
	}{
		{name: "new", to: models.CUSTOMER_STATUS_NEW, want: customerv1.CustomerStatus_NEW},
		{name: "profile_filled", to: models.CUSTOMER_STATUS_PROFILE_FILLED, want: customerv1.CustomerStatus_PROFILE_FILLED},
		{name: "on_kyc", to: models.CUSTOMER_STATUS_ON_KYC, want: customerv1.CustomerStatus_ON_KYC},
		{name: "active", to: models.CUSTOMER_STATUS_ACTIVE, want: customerv1.CustomerStatus_ACTIVE},
		{name: "rejected", to: models.CUSTOMER_STATUS_REJECTED, want: customerv1.CustomerStatus_REJECTED},
		{name: "blocked", to: models.CUSTOMER_STATUS_BLOCKED, want: customerv1.CustomerStatus_BLOCKED},
		{name: "unspecified", to: models.CUSTOMER_STATUS_UNSPECIFIED, want: customerv1.CustomerStatus_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statuses := &fakeCustomerStatusServiceClient{}
			customerClient := NewCustomerClient(&fakeCustomerServiceClient{}, statuses)

			err := customerClient.SetStatus(t.Context(), TEST_USER_ID, tt.to, TEST_REASON, nil)

			require.NoError(t, err)
			require.Len(t, statuses.setStatusCalls, 1)
			assert.Equal(t, tt.want, statuses.setStatusCalls[0].GetStatus())
		})
	}
}
