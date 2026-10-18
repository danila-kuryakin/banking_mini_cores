package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/mocks"
)

var (
	testUserID     = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testActorID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testCustomerID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

// testTime - фиксированное время для моделей из моков.
var testTime = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

// errDB - произвольная ошибка репозитория.
var errDB = errors.New("db is down")

type testEnv struct {
	service *CustomerService
	repo    *mocks.CustomerRepositoryMock
}

// newTestEnv собирает CustomerService над свежим моком репозитория.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	repo := &mocks.CustomerRepositoryMock{}
	t.Cleanup(func() { repo.AssertExpectations(t) })

	return &testEnv{
		service: NewCustomerService(&repository.Repository{Customer: repo}),
		repo:    repo,
	}
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

func statusPtr(s models.Status) *models.Status {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
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

func (e *testEnv) expectStatus(current models.Status) {
	e.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(statusPtr(current), timePtr(testTime), nil).Once()
}

func TestUpdateProfile_InvalidProfile(t *testing.T) {
	tomorrow := time.Now().UTC().AddDate(0, 0, 2)

	tests := []struct {
		name    string
		mutate  func(p *models.Profile)
		wantErr error
	}{
		{name: "неверный телефон", mutate: func(p *models.Profile) { p.Phone = "abc" }, wantErr: domain.ErrPhoneIsNotValid},
		{name: "дата рождения завтра", mutate: func(p *models.Profile) { p.BirthDate = &tomorrow }, wantErr: domain.ErrBirthDateInFuture},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			profile := fullProfile()
			tc.mutate(&profile)

			_, err := env.service.UpdateProfile(context.Background(), testUserID, profile)

			require.ErrorIs(t, err, tc.wantErr)
			env.repo.AssertNotCalled(t, "GetStatusByUserID", mock.Anything, mock.Anything)
			env.repo.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateProfile_FormatCheckedBeforeStatus(t *testing.T) {
	env := newTestEnv(t)
	// Статус on_kyc не задаём: формат должен отбить запрос до чтения статуса.
	profile := fullProfile()
	profile.Phone = "abc"

	_, err := env.service.UpdateProfile(context.Background(), testUserID, profile)

	require.ErrorIs(t, err, domain.ErrPhoneIsNotValid)
	require.NotErrorIs(t, err, domain.ErrProfileLocked)
}

func TestUpdateProfile_CustomerNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(nil, nil, domain.ErrCustomerNotFound).Once()

	_, err := env.service.UpdateProfile(context.Background(), testUserID, fullProfile())

	require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	env.repo.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProfile_Locked(t *testing.T) {
	for _, current := range []models.Status{models.STATUS_ON_KYC, models.STATUS_ACTIVE, models.STATUS_REJECTED, models.STATUS_BLOCKED} {
		t.Run(string(current), func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(current)

			_, err := env.service.UpdateProfile(context.Background(), testUserID, fullProfile())

			require.ErrorIs(t, err, domain.ErrProfileLocked)
			env.repo.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateProfile_Incomplete(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(p *models.Profile)
	}{
		{name: "нет телефона", mutate: func(p *models.Profile) { p.Phone = "" }},
		{name: "имя из пробелов", mutate: func(p *models.Profile) { p.FirstName = "   " }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(models.STATUS_PROFILE_FILLED)
			profile := fullProfile()
			tc.mutate(&profile)

			_, err := env.service.UpdateProfile(context.Background(), testUserID, profile)

			require.ErrorIs(t, err, domain.ErrProfileIncomplete)
			env.repo.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateProfile_NextStatus(t *testing.T) {
	partial := fullProfile()
	partial.Phone = ""

	tests := []struct {
		name     string
		current  models.Status
		profile  models.Profile
		wantNext models.Status
	}{
		{name: "новый, анкета неполная", current: models.STATUS_NEW, profile: partial, wantNext: models.STATUS_NEW},
		{name: "новый, анкета полная", current: models.STATUS_NEW, profile: fullProfile(), wantNext: models.STATUS_PROFILE_FILLED},
		{name: "заполнена, анкета полная", current: models.STATUS_PROFILE_FILLED, profile: fullProfile(), wantNext: models.STATUS_PROFILE_FILLED},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(tc.current)
			customer := testCustomer(tc.wantNext)
			env.repo.On("UpdateProfile", mock.Anything, testUserID, tc.profile, tc.current, tc.wantNext).Return(customer, nil).Once()

			got, err := env.service.UpdateProfile(context.Background(), testUserID, tc.profile)

			require.NoError(t, err)
			assert.Same(t, customer, got)
		})
	}
}

func TestUpdateProfile_NormalizesBeforeSave(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)

	raw := fullProfile()
	raw.FirstName = "  Иван "
	raw.LastName = "\tПетров\n"
	raw.Citizenship = " RU"
	raw.Phone = "+79001234567  "

	want := fullProfile()
	want.BirthDate = raw.BirthDate

	customer := testCustomer(models.STATUS_PROFILE_FILLED)
	env.repo.On("UpdateProfile", mock.Anything, testUserID, want, models.STATUS_NEW, models.STATUS_PROFILE_FILLED).Return(customer, nil).Once()

	got, err := env.service.UpdateProfile(context.Background(), testUserID, raw)

	require.NoError(t, err)
	assert.Same(t, customer, got)
}

func TestUpdateProfile_StatusConflict(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)
	env.repo.On("UpdateProfile", mock.Anything, testUserID, fullProfile(), models.STATUS_NEW, models.STATUS_PROFILE_FILLED).
		Return(nil, domain.ErrStatusConflict).Once()

	_, err := env.service.UpdateProfile(context.Background(), testUserID, fullProfile())

	require.ErrorIs(t, err, domain.ErrStatusConflict)
}

func TestSetStatus_InvalidTarget(t *testing.T) {
	for _, to := range []models.Status{models.STATUS_UNSPECIFIED, models.Status("foo")} {
		t.Run(string(to), func(t *testing.T) {
			env := newTestEnv(t)

			_, err := env.service.SetStatus(context.Background(), testUserID, to, "", nil)

			require.ErrorIs(t, err, domain.ErrStatusRequired)
		})
	}
}

func TestSetStatus_CustomerNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(nil, nil, domain.ErrCustomerNotFound).Once()

	_, err := env.service.SetStatus(context.Background(), testUserID, models.STATUS_ACTIVE, "", nil)

	require.ErrorIs(t, err, domain.ErrCustomerNotFound)
}

func TestSetStatus_SameStatusIsNoop(t *testing.T) {
	// Повтор успешен раньше проверок причины и blocked.
	for _, current := range []models.Status{models.STATUS_ACTIVE, models.STATUS_REJECTED, models.STATUS_BLOCKED} {
		t.Run(string(current), func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(current)

			got, err := env.service.SetStatus(context.Background(), testUserID, current, "", nil)

			require.NoError(t, err)
			assert.Equal(t, &models.StatusChange{Previous: current, Current: current, ChangedAt: testTime}, got)
			env.repo.AssertNotCalled(t, "SetStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestSetStatus_BlockingNotImplemented(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_ON_KYC)

	_, err := env.service.SetStatus(context.Background(), testUserID, models.STATUS_BLOCKED, "fraud", nil)

	require.ErrorIs(t, err, domain.ErrBlockingNotImplemented)
	env.repo.AssertNotCalled(t, "SetStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSetStatus_ReasonRequired(t *testing.T) {
	tests := []struct {
		name    string
		current models.Status
		reason  string
	}{
		{name: "на kyc, пустая причина", current: models.STATUS_ON_KYC, reason: ""},
		{name: "на kyc, причина из пробелов", current: models.STATUS_ON_KYC, reason: "   "},
		// Причина проверяется раньше таблицы переходов: new -> rejected запрещён.
		{name: "новый, пустая причина", current: models.STATUS_NEW, reason: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(tc.current)

			_, err := env.service.SetStatus(context.Background(), testUserID, models.STATUS_REJECTED, tc.reason, nil)

			require.ErrorIs(t, err, domain.ErrReasonRequired)
		})
	}
}

func TestSetStatus_TransitionForbidden(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_NEW)

	_, err := env.service.SetStatus(context.Background(), testUserID, models.STATUS_ACTIVE, "", nil)

	require.ErrorIs(t, err, domain.ErrStatusTransitionForbidden)
	assert.Contains(t, err.Error(), "new -> active")
}

func TestSetStatus_PassesToRepository(t *testing.T) {
	actorID := testActorID

	tests := []struct {
		name            string
		to              models.Status
		reason          string
		actorID         *uuid.UUID
		wantAllowedFrom []models.Status
	}{
		{name: "активен без инициатора", to: models.STATUS_ACTIVE, wantAllowedFrom: []models.Status{models.STATUS_ON_KYC}},
		{name: "отказ с причиной и инициатором", to: models.STATUS_REJECTED, reason: "fake passport", actorID: &actorID, wantAllowedFrom: models.AllowedFrom(models.STATUS_REJECTED)},
		{name: "возврат к анкете", to: models.STATUS_PROFILE_FILLED, wantAllowedFrom: []models.Status{models.STATUS_NEW, models.STATUS_ON_KYC}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.expectStatus(models.STATUS_ON_KYC)
			change := &models.StatusChange{Previous: models.STATUS_ON_KYC, Current: tc.to, ChangedAt: testTime}
			env.repo.On("SetStatus", mock.Anything, testUserID, tc.to, tc.reason, tc.actorID, tc.wantAllowedFrom).Return(change, nil).Once()

			got, err := env.service.SetStatus(context.Background(), testUserID, tc.to, tc.reason, tc.actorID)

			require.NoError(t, err)
			assert.Same(t, change, got)
		})
	}
}

func TestSetStatus_StatusConflict(t *testing.T) {
	env := newTestEnv(t)
	env.expectStatus(models.STATUS_ON_KYC)
	env.repo.On("SetStatus", mock.Anything, testUserID, models.STATUS_ACTIVE, "", (*uuid.UUID)(nil), []models.Status{models.STATUS_ON_KYC}).
		Return(nil, domain.ErrStatusConflict).Once()

	_, err := env.service.SetStatus(context.Background(), testUserID, models.STATUS_ACTIVE, "", nil)

	require.ErrorIs(t, err, domain.ErrStatusConflict)
}

func TestListCustomers_Page(t *testing.T) {
	tests := []struct {
		name                  string
		limit, offset         int
		wantLimit, wantOffset int
	}{
		{name: "умолчания", limit: 0, offset: 0, wantLimit: 20, wantOffset: 0},
		{name: "отрицательный limit", limit: -5, offset: 0, wantLimit: 20, wantOffset: 0},
		{name: "как есть", limit: 50, offset: 7, wantLimit: 50, wantOffset: 7},
		{name: "потолок", limit: 100, offset: 0, wantLimit: 100, wantOffset: 0},
		{name: "выше потолка", limit: 101, offset: 0, wantLimit: 100, wantOffset: 0},
		{name: "отрицательный offset", limit: 10, offset: -1, wantLimit: 10, wantOffset: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			customers := []*models.Customer{testCustomer(models.STATUS_NEW)}
			env.repo.On("ListCustomers", mock.Anything, tc.wantLimit, tc.wantOffset).Return(customers, nil).Once()

			got, err := env.service.ListCustomers(context.Background(), tc.limit, tc.offset)

			require.NoError(t, err)
			assert.Equal(t, customers, got)
		})
	}
}

func TestListCustomers_Error(t *testing.T) {
	env := newTestEnv(t)
	env.repo.On("ListCustomers", mock.Anything, 20, 0).Return(nil, errDB).Once()

	_, err := env.service.ListCustomers(context.Background(), 0, 0)

	require.ErrorIs(t, err, errDB)
}

func TestCreateProfile(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		env := newTestEnv(t)
		customer := testCustomer(models.STATUS_NEW)
		env.repo.On("CreateCustomer", mock.Anything, testUserID, models.STATUS_NEW).Return(customer, nil).Once()

		got, err := env.service.CreateProfile(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Same(t, customer, got)
	})

	t.Run("ошибка", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("CreateCustomer", mock.Anything, testUserID, models.STATUS_NEW).Return(nil, domain.ErrCustomerExists).Once()

		_, err := env.service.CreateProfile(context.Background(), testUserID)

		require.ErrorIs(t, err, domain.ErrCustomerExists)
	})
}

func TestGetCustomer(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		env := newTestEnv(t)
		customer := testCustomer(models.STATUS_NEW)
		env.repo.On("GetCustomer", mock.Anything, testUserID).Return(customer, nil).Once()

		got, err := env.service.GetCustomer(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Same(t, customer, got)
	})

	t.Run("ошибка", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("GetCustomer", mock.Anything, testUserID).Return(nil, domain.ErrCustomerNotFound).Once()

		_, err := env.service.GetCustomer(context.Background(), testUserID)

		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}

func TestGetCustomerStatus(t *testing.T) {
	t.Run("успех", func(t *testing.T) {
		env := newTestEnv(t)
		customerStatus := statusPtr(models.STATUS_ACTIVE)
		changedAt := timePtr(testTime)
		env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(customerStatus, changedAt, nil).Once()

		gotStatus, gotChangedAt, err := env.service.GetCustomerStatus(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Same(t, customerStatus, gotStatus)
		assert.Same(t, changedAt, gotChangedAt)
	})

	t.Run("ошибка", func(t *testing.T) {
		env := newTestEnv(t)
		env.repo.On("GetStatusByUserID", mock.Anything, testUserID).Return(nil, nil, domain.ErrCustomerNotFound).Once()

		_, _, err := env.service.GetCustomerStatus(context.Background(), testUserID)

		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}
