package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// Идентификаторы и данные анкеты для тестов
var (
	TEST_USER_ID                 = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_OTHER_USER_ID           = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TEST_APPLICATION_ID          = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	TEST_PREVIOUS_APPLICATION_ID = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	TEST_OFFICER_ID              = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	TEST_SANCTION_ID             = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

const (
	TEST_FIRST_NAME = "Ivan"
	TEST_LAST_NAME  = "Petrov"
)

// Текст произвольной ошибки зависимости
const TEST_BOOM_MESSAGE = "boom"

// testBirthDate возвращает фиксированную дату рождения из примеров спецификации
func testBirthDate() *time.Time {
	birthDate := time.Date(1990, time.May, 15, 0, 0, 0, 0, time.UTC)
	return &birthDate
}

// errNotUsed - метод, который сервис вызывать не должен
var errNotUsed = errors.New("method must not be called")

// transitionCall - аргументы одного вызова Transition
type transitionCall struct {
	ApplicationID uuid.UUID
	Transition    models.Transition
	AllowedFrom   []models.ApplicationStatus
}

// claimCall - аргументы одного вызова Claim
type claimCall struct {
	ApplicationID uuid.UUID
	OfficerID     uuid.UUID
}

// fakeApplicationRepository - in-memory хранилище заявок
type fakeApplicationRepository struct {
	apps map[uuid.UUID]models.Application

	createID  uuid.UUID
	createErr error

	getByIDErr error

	latest    *models.Application
	latestErr error

	listResult []*models.Application
	listErr    error

	transitionErr error

	claimResult *models.Application
	claimErr    error

	createCalls     []models.Application
	getByIDCalls    []uuid.UUID
	latestCalls     []uuid.UUID
	listCalls       []models.ApplicationFilter
	transitionCalls []transitionCall
	claimCalls      []claimCall
}

var _ interfaces.ApplicationRepository = (*fakeApplicationRepository)(nil)

func newFakeApplicationRepository(apps ...models.Application) *fakeApplicationRepository {
	repo := &fakeApplicationRepository{apps: map[uuid.UUID]models.Application{}, createID: TEST_APPLICATION_ID}
	for _, app := range apps {
		repo.apps[app.ID] = app
	}
	return repo
}

func (f *fakeApplicationRepository) Create(_ context.Context, app models.Application) (*models.Application, error) {
	f.createCalls = append(f.createCalls, app)
	if f.createErr != nil {
		return nil, f.createErr
	}
	app.ID = f.createID
	app.Status = models.STATUS_SUBMITTED
	f.apps[app.ID] = app
	return &app, nil
}

func (f *fakeApplicationRepository) GetByID(_ context.Context, applicationID uuid.UUID) (*models.Application, error) {
	f.getByIDCalls = append(f.getByIDCalls, applicationID)
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	app, ok := f.apps[applicationID]
	if !ok {
		return nil, domain.ErrApplicationNotFound
	}
	return &app, nil
}

func (f *fakeApplicationRepository) GetLatestByUserID(_ context.Context, userID uuid.UUID) (*models.Application, error) {
	f.latestCalls = append(f.latestCalls, userID)
	if f.latestErr != nil {
		return nil, f.latestErr
	}
	if f.latest == nil {
		return nil, domain.ErrApplicationNotFound
	}
	app := *f.latest
	return &app, nil
}

func (f *fakeApplicationRepository) List(_ context.Context, filter models.ApplicationFilter) ([]*models.Application, error) {
	f.listCalls = append(f.listCalls, filter)
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeApplicationRepository) Transition(_ context.Context, applicationID uuid.UUID, t models.Transition, allowedFrom []models.ApplicationStatus) (*models.Application, error) {
	f.transitionCalls = append(f.transitionCalls, transitionCall{ApplicationID: applicationID, Transition: t, AllowedFrom: allowedFrom})
	if f.transitionErr != nil {
		return nil, f.transitionErr
	}
	app, ok := f.apps[applicationID]
	if !ok {
		return nil, domain.ErrApplicationNotFound
	}
	if !slices.Contains(allowedFrom, app.Status) {
		return nil, domain.ErrStatusConflict
	}
	app.Status = t.To
	app.Reason = t.Reason
	app.RequestedTypes = t.RequestedTypes
	f.apps[applicationID] = app
	return &app, nil
}

func (f *fakeApplicationRepository) TakeForAutoChecks(_ context.Context, _ int) ([]*models.Application, error) {
	return nil, errNotUsed
}

func (f *fakeApplicationRepository) Claim(_ context.Context, applicationID, officerID uuid.UUID) (*models.Application, error) {
	f.claimCalls = append(f.claimCalls, claimCall{ApplicationID: applicationID, OfficerID: officerID})
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return f.claimResult, nil
}

// pageCall - аргументы постраничного чтения
type pageCall struct {
	Limit  int
	Offset int
}

// fakeSanctionsRepository - санкционный список с настраиваемыми ответами
type fakeSanctionsRepository struct {
	addResult *models.SanctionEntry
	addErr    error

	listResult []models.SanctionEntry
	listErr    error

	addCalls  []models.SanctionEntry
	listCalls []pageCall
}

var _ interfaces.SanctionsRepository = (*fakeSanctionsRepository)(nil)

func (f *fakeSanctionsRepository) FindMatch(_ context.Context, _, _ string, _ time.Time) (*models.SanctionEntry, error) {
	return nil, errNotUsed
}

func (f *fakeSanctionsRepository) Add(_ context.Context, entry models.SanctionEntry) (*models.SanctionEntry, error) {
	f.addCalls = append(f.addCalls, entry)
	if f.addErr != nil {
		return nil, f.addErr
	}
	if f.addResult != nil {
		return f.addResult, nil
	}
	return &entry, nil
}

func (f *fakeSanctionsRepository) List(_ context.Context, limit, offset int) ([]models.SanctionEntry, error) {
	f.listCalls = append(f.listCalls, pageCall{Limit: limit, Offset: offset})
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

// setStatusCall - аргументы одного вызова SetStatus
type setStatusCall struct {
	UserID  uuid.UUID
	To      models.CustomerStatus
	Reason  string
	ActorID *uuid.UUID
}

// fakeCustomerProfiles - customer-service с настраиваемыми ответами
type fakeCustomerProfiles struct {
	profile       *models.CustomerProfile
	getProfileErr error
	setStatusErr  error

	getProfileCalls []uuid.UUID
	setStatusCalls  []setStatusCall
}

var _ interfaces.CustomerProfiles = (*fakeCustomerProfiles)(nil)

func (f *fakeCustomerProfiles) GetProfile(_ context.Context, userID uuid.UUID) (*models.CustomerProfile, error) {
	f.getProfileCalls = append(f.getProfileCalls, userID)
	if f.getProfileErr != nil {
		return nil, f.getProfileErr
	}
	if f.profile == nil {
		return nil, domain.ErrCustomerNotFound
	}
	profile := *f.profile
	return &profile, nil
}

func (f *fakeCustomerProfiles) SetStatus(_ context.Context, userID uuid.UUID, to models.CustomerStatus, reason string, actorID *uuid.UUID) error {
	f.setStatusCalls = append(f.setStatusCalls, setStatusCall{UserID: userID, To: to, Reason: reason, ActorID: actorID})
	return f.setStatusErr
}

// fakeDocuments - filestore с настраиваемыми ответами
type fakeDocuments struct {
	hasRequired    bool
	missing        []models.DocumentType
	hasRequiredErr error

	confirmed    []models.DocumentType
	confirmedErr error

	hasRequiredCalls []uuid.UUID
	confirmedCalls   []uuid.UUID
}

var _ interfaces.Documents = (*fakeDocuments)(nil)

func (f *fakeDocuments) HasRequiredDocuments(_ context.Context, userID uuid.UUID) (bool, []models.DocumentType, error) {
	f.hasRequiredCalls = append(f.hasRequiredCalls, userID)
	if f.hasRequiredErr != nil {
		return false, nil, f.hasRequiredErr
	}
	return f.hasRequired, f.missing, nil
}

func (f *fakeDocuments) ConfirmedTypes(_ context.Context, userID uuid.UUID) ([]models.DocumentType, error) {
	f.confirmedCalls = append(f.confirmedCalls, userID)
	if f.confirmedErr != nil {
		return nil, f.confirmedErr
	}
	return f.confirmed, nil
}
