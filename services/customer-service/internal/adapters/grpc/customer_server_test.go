package grpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/mocks"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

const (
	TEST_USER_ID     = "11111111-1111-1111-1111-111111111111"
	TEST_ACTOR_ID    = "22222222-2222-2222-2222-222222222222"
	TEST_CUSTOMER_ID = "33333333-3333-3333-3333-333333333333"
)

var (
	testUserID     = uuid.MustParse(TEST_USER_ID)
	testActorID    = uuid.MustParse(TEST_ACTOR_ID)
	testCustomerID = uuid.MustParse(TEST_CUSTOMER_ID)
)

// testTime - фиксированное время для моделей из моков.
var testTime = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

// errDB - произвольная ошибка репозитория.
var errDB = errors.New("db is down")

type testEnv struct {
	customers *CustomerServer
	statuses  *StatusServer
	repo      *mocks.CustomerRepositoryMock
}

// newTestEnv собирает оба сервера над настоящим сервисом и моком репозитория.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	repo := &mocks.CustomerRepositoryMock{}
	t.Cleanup(func() { repo.AssertExpectations(t) })

	svc := service.NewService(&repository.Repository{Customer: repo})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &testEnv{
		customers: NewCustomerServer(svc, logger),
		statuses:  NewStatusServer(svc, logger),
		repo:      repo,
	}
}

func (e *testEnv) expectStatus(current models.Status) {
	e.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(&current, &testTime, nil).Once()
}

// fullProfile - полная валидная анкета с датой рождения 30 лет назад.
func fullProfile() models.Profile {
	now := time.Now().UTC()
	birthDate := time.Date(now.Year()-30, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return models.Profile{
		FirstName:   "Иван",
		LastName:    "Петров",
		BirthDate:   &birthDate,
		Citizenship: "RU",
		Phone:       "+79001234567",
	}
}

func fullProfileProto() *customerv1.Profile {
	return profileToProto(fullProfile())
}

func testCustomer(customerStatus models.Status) *models.Customer {
	return &models.Customer{
		ID:              testCustomerID,
		UserID:          testUserID,
		Status:          customerStatus,
		StatusChangedAt: testTime,
		Profile:         fullProfile(),
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}
}

// requireStatusErr проверяет код gRPC и совпадение с ожидаемой ошибкой.
func requireStatusErr(t *testing.T, err error, code codes.Code, want error) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, code, status.Code(err))
	require.ErrorIs(t, err, want)
}

func TestCreateProfile_BadUserID(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.customers.CreateProfile(context.Background(), &customerv1.CreateProfileRequest{UserId: "bad"})

	requireStatusErr(t, err, codes.InvalidArgument, domain.ErrUserIDMustBeUUID)
}

func TestCreateProfile_Errors(t *testing.T) {
	tests := []struct {
		name     string
		repoErr  error
		wantCode codes.Code
		wantErr  error
	}{
		{name: "уже есть", repoErr: domain.ErrCustomerExists, wantCode: codes.AlreadyExists, wantErr: domain.ErrCustomerAlreadyExists},
		{name: "сбой базы", repoErr: errDB, wantCode: codes.Internal, wantErr: domain.ErrCreateProfileFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.repo.On("CreateCustomer", mock.Anything, testUserID, models.STATUS_NEW).Return(nil, tc.repoErr).Once()

			_, err := env.customers.CreateProfile(context.Background(), &customerv1.CreateProfileRequest{UserId: TEST_USER_ID})

			requireStatusErr(t, err, tc.wantCode, tc.wantErr)
		})
	}
}

func TestCreateProfile_Success(t *testing.T) {
	env := newTestEnv(t)
	customer := testCustomer(models.STATUS_NEW)
	env.repo.On("CreateCustomer", mock.Anything, testUserID, models.STATUS_NEW).Return(customer, nil).Once()

	got, err := env.customers.CreateProfile(context.Background(), &customerv1.CreateProfileRequest{UserId: TEST_USER_ID})

	require.NoError(t, err)
	assert.Equal(t, TEST_CUSTOMER_ID, got.Id)
	assert.Equal(t, TEST_USER_ID, got.UserId)
	assert.Equal(t, customerv1.CustomerStatus_NEW, got.Status)
	assert.Equal(t, testTime, got.StatusChangedAt.AsTime())
	assert.Equal(t, testTime, got.CreatedAt.AsTime())
	assert.Equal(t, testTime, got.UpdatedAt.AsTime())
}

func TestGetCustomer(t *testing.T) {
	t.Run("не найден", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("GetCustomer", mock.Anything, testUserID).Return(nil, domain.ErrCustomerNotFound).Once()

		_, err := env.customers.GetCustomer(context.Background(), &customerv1.GetCustomerRequest{UserId: TEST_USER_ID})

		requireStatusErr(t, err, codes.NotFound, domain.ErrCustomerMissing)
	})

	t.Run("сбой базы", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("GetCustomer", mock.Anything, testUserID).Return(nil, errDB).Once()

		_, err := env.customers.GetCustomer(context.Background(), &customerv1.GetCustomerRequest{UserId: TEST_USER_ID})

		require.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("успех", func(t *testing.T) {
		env := newTestEnv(t)
		customer := testCustomer(models.STATUS_ACTIVE)
		env.repo.On("GetCustomer", mock.Anything, testUserID).Return(customer, nil).Once()

		got, err := env.customers.GetCustomer(context.Background(), &customerv1.GetCustomerRequest{UserId: TEST_USER_ID})

		require.NoError(t, err)
		assert.Equal(t, customerToProto(customer), got)
	})
}

func TestUpdateProfile_RejectedBeforeService(t *testing.T) {
	tests := []struct {
		name     string
		profile  *customerv1.Profile
		wantCode codes.Code
		wantErr  error
	}{
		{name: "нет анкеты", profile: nil, wantCode: codes.InvalidArgument, wantErr: domain.ErrProfileRequired},
		{name: "29 февраля не високосного", profile: &customerv1.Profile{BirthDate: &date.Date{Year: 2001, Month: 2, Day: 29}}, wantCode: codes.InvalidArgument, wantErr: domain.ErrBirthDateIsNotValid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)

			_, err := env.customers.UpdateProfile(context.Background(), &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID, Profile: tc.profile})

			requireStatusErr(t, err, tc.wantCode, tc.wantErr)
		})
	}
}

func TestUpdateProfile_LeapDay(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)

	leapDay := time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC)
	want := fullProfile()
	want.BirthDate = &leapDay
	customer := testCustomer(models.STATUS_PROFILE_FILLED)
	env.repo.On("UpdateProfile", mock.Anything, testUserID, want, models.STATUS_NEW, models.STATUS_PROFILE_FILLED).Return(customer, nil).Once()

	in := fullProfileProto()
	in.BirthDate = &date.Date{Year: 2000, Month: 2, Day: 29}

	_, err := env.customers.UpdateProfile(context.Background(), &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID, Profile: in})

	require.NoError(t, err)
}

func TestUpdateProfile_Errors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(env *testEnv)
		profile  func() *customerv1.Profile
		wantCode codes.Code
		wantErr  error
	}{
		{
			name: "клиент не найден",
			setup: func(env *testEnv) {
				env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(nil, nil, domain.ErrCustomerNotFound).Once()
			},
			profile:  fullProfileProto,
			wantCode: codes.NotFound,
			wantErr:  domain.ErrCustomerMissing,
		},
		{
			name:     "анкета закрыта",
			setup:    func(env *testEnv) { env.expectStatus(models.STATUS_ON_KYC) },
			profile:  fullProfileProto,
			wantCode: codes.FailedPrecondition,
			wantErr:  domain.ErrProfileLocked,
		},
		{
			name: "статус сменился",
			setup: func(env *testEnv) {
				env.expectStatus(models.STATUS_NEW)
				env.repo.On("UpdateProfile", mock.Anything, testUserID, fullProfile(), models.STATUS_NEW, models.STATUS_PROFILE_FILLED).
					Return(nil, domain.ErrStatusConflict).Once()
			},
			profile:  fullProfileProto,
			wantCode: codes.Aborted,
			wantErr:  domain.ErrStatusChanged,
		},
		{
			name:  "неверный телефон",
			setup: func(env *testEnv) {},
			profile: func() *customerv1.Profile {
				in := fullProfileProto()
				in.Phone = "abc"
				return in
			},
			wantCode: codes.InvalidArgument,
			wantErr:  domain.ErrPhoneIsNotValid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			tc.setup(env)

			_, err := env.customers.UpdateProfile(context.Background(), &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID, Profile: tc.profile()})

			requireStatusErr(t, err, tc.wantCode, tc.wantErr)
		})
	}
}

func TestUpdateProfile_InternalError(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)
	env.repo.On("UpdateProfile", mock.Anything, testUserID, fullProfile(), models.STATUS_NEW, models.STATUS_PROFILE_FILLED).Return(nil, errDB).Once()

	_, err := env.customers.UpdateProfile(context.Background(), &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID, Profile: fullProfileProto()})

	require.Equal(t, codes.Internal, status.Code(err))
}

func TestUpdateProfile_Success(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)
	customer := testCustomer(models.STATUS_PROFILE_FILLED)
	env.repo.On("UpdateProfile", mock.Anything, testUserID, fullProfile(), models.STATUS_NEW, models.STATUS_PROFILE_FILLED).Return(customer, nil).Once()

	got, err := env.customers.UpdateProfile(context.Background(), &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID, Profile: fullProfileProto()})

	require.NoError(t, err)
	assert.Equal(t, customerToProto(customer), got)
}

func TestGetCustomerStatus(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		env := newTestEnv(t)
		env.expectStatus(models.STATUS_ON_KYC)

		got, err := env.customers.GetCustomerStatus(context.Background(), &customerv1.GetCustomerStatusRequest{UserId: TEST_USER_ID})

		require.NoError(t, err)
		assert.Equal(t, customerv1.CustomerStatus_ON_KYC, got.Status)
		assert.Equal(t, testTime, got.StatusChangedAt.AsTime())
	})

	t.Run("не найден", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(nil, nil, domain.ErrCustomerNotFound).Once()

		_, err := env.customers.GetCustomerStatus(context.Background(), &customerv1.GetCustomerStatusRequest{UserId: TEST_USER_ID})

		requireStatusErr(t, err, codes.NotFound, domain.ErrCustomerMissing)
	})
}

func TestListCustomers(t *testing.T) {
	t.Run("сбой базы", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("ListCustomers", mock.Anything, 20, 0).Return(nil, errDB).Once()

		_, err := env.customers.ListCustomers(context.Background(), &customerv1.ListCustomersRequest{})

		requireStatusErr(t, err, codes.Internal, domain.ErrListCustomersFailed)
	})

	t.Run("две карточки", func(t *testing.T) {
		env := newTestEnv(t)
		first := testCustomer(models.STATUS_NEW)
		second := testCustomer(models.STATUS_ACTIVE)
		second.ID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
		second.UserID = uuid.MustParse("55555555-5555-5555-5555-555555555555")
		env.repo.On("ListCustomers", mock.Anything, 10, 5).Return([]*models.Customer{first, second}, nil).Once()

		got, err := env.customers.ListCustomers(context.Background(), &customerv1.ListCustomersRequest{Limit: 10, Offset: 5})

		require.NoError(t, err)
		require.Len(t, got.Customers, 2)
		assert.Equal(t, customerToProto(first), got.Customers[0])
		assert.Equal(t, customerToProto(second), got.Customers[1])
	})

	t.Run("пусто", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("ListCustomers", mock.Anything, 20, 0).Return([]*models.Customer{}, nil).Once()

		got, err := env.customers.ListCustomers(context.Background(), &customerv1.ListCustomersRequest{})

		require.NoError(t, err)
		assert.Empty(t, got.Customers)
	})
}
