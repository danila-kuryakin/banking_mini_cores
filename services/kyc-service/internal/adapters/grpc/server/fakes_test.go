package server

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// Проверки реализации интерфейсов на этапе компиляции
var (
	_ interfaces.ApplicationRepository = (*fakeApplicationRepository)(nil)
	_ interfaces.SanctionsRepository   = (*fakeSanctionsRepository)(nil)
	_ interfaces.CustomerProfiles      = (*fakeCustomerProfiles)(nil)
	_ interfaces.Documents             = (*fakeDocuments)(nil)
)

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

// pageCall - аргументы одного вызова List санкционного списка
type pageCall struct {
	Limit  int
	Offset int
}

// fakeApplicationRepository - фейк репозитория заявок с настраиваемыми ответами и журналом вызовов
type fakeApplicationRepository struct {
	createResult     *models.Application
	createErr        error
	getByIDResult    *models.Application
	getByIDErr       error
	latestResult     *models.Application
	latestErr        error
	listResult       []*models.Application
	listErr          error
	transitionResult *models.Application
	transitionErr    error
	takeErr          error
	claimResult      *models.Application
	claimErr         error

	createCalls     []models.Application
	getByIDCalls    []uuid.UUID
	latestCalls     []uuid.UUID
	listCalls       []models.ApplicationFilter
	transitionCalls []transitionCall
	takeCalls       []int
	claimCalls      []claimCall
}

func (f *fakeApplicationRepository) Create(_ context.Context, app models.Application) (*models.Application, error) {
	f.createCalls = append(f.createCalls, app)
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &app, nil
}

func (f *fakeApplicationRepository) GetByID(_ context.Context, applicationID uuid.UUID) (*models.Application, error) {
	f.getByIDCalls = append(f.getByIDCalls, applicationID)
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	if f.getByIDResult != nil {
		return f.getByIDResult, nil
	}
	return &models.Application{}, nil
}

func (f *fakeApplicationRepository) GetLatestByUserID(_ context.Context, userID uuid.UUID) (*models.Application, error) {
	f.latestCalls = append(f.latestCalls, userID)
	if f.latestErr != nil {
		return nil, f.latestErr
	}
	if f.latestResult != nil {
		return f.latestResult, nil
	}
	return &models.Application{}, nil
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
	if f.transitionResult != nil {
		return f.transitionResult, nil
	}
	return &models.Application{}, nil
}

func (f *fakeApplicationRepository) TakeForAutoChecks(_ context.Context, limit int) ([]*models.Application, error) {
	f.takeCalls = append(f.takeCalls, limit)
	if f.takeErr != nil {
		return nil, f.takeErr
	}
	return nil, nil
}

func (f *fakeApplicationRepository) Claim(_ context.Context, applicationID, officerID uuid.UUID) (*models.Application, error) {
	f.claimCalls = append(f.claimCalls, claimCall{ApplicationID: applicationID, OfficerID: officerID})
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if f.claimResult != nil {
		return f.claimResult, nil
	}
	return &models.Application{}, nil
}

// callCount - общее число вызовов фейка
func (f *fakeApplicationRepository) callCount() int {
	return len(f.createCalls) + len(f.getByIDCalls) + len(f.latestCalls) + len(f.listCalls) +
		len(f.transitionCalls) + len(f.takeCalls) + len(f.claimCalls)
}

// fakeSanctionsRepository - фейк санкционного списка
type fakeSanctionsRepository struct {
	addResult  *models.SanctionEntry
	addErr     error
	listResult []models.SanctionEntry
	listErr    error

	findCalls int
	addCalls  []models.SanctionEntry
	listCalls []pageCall
}

func (f *fakeSanctionsRepository) FindMatch(_ context.Context, _, _ string, _ time.Time) (*models.SanctionEntry, error) {
	f.findCalls++
	return nil, nil
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

// callCount - общее число вызовов фейка
func (f *fakeSanctionsRepository) callCount() int {
	return f.findCalls + len(f.addCalls) + len(f.listCalls)
}

// setStatusCall - аргументы одного вызова SetStatus
type setStatusCall struct {
	UserID  uuid.UUID
	To      models.CustomerStatus
	Reason  string
	ActorID *uuid.UUID
}

// fakeCustomerProfiles - фейк клиента customer-service
type fakeCustomerProfiles struct {
	profileResult *models.CustomerProfile
	getProfileErr error
	setStatusErr  error

	getProfileCalls []uuid.UUID
	setStatusCalls  []setStatusCall
}

func (f *fakeCustomerProfiles) GetProfile(_ context.Context, userID uuid.UUID) (*models.CustomerProfile, error) {
	f.getProfileCalls = append(f.getProfileCalls, userID)
	if f.getProfileErr != nil {
		return nil, f.getProfileErr
	}
	if f.profileResult != nil {
		return f.profileResult, nil
	}
	return &models.CustomerProfile{}, nil
}

func (f *fakeCustomerProfiles) SetStatus(_ context.Context, userID uuid.UUID, to models.CustomerStatus, reason string, actorID *uuid.UUID) error {
	f.setStatusCalls = append(f.setStatusCalls, setStatusCall{UserID: userID, To: to, Reason: reason, ActorID: actorID})
	return f.setStatusErr
}

// callCount - общее число вызовов фейка
func (f *fakeCustomerProfiles) callCount() int {
	return len(f.getProfileCalls) + len(f.setStatusCalls)
}

// fakeDocuments - фейк клиента filestore-service
type fakeDocuments struct {
	hasRequired    bool
	missing        []models.DocumentType
	hasRequiredErr error
	confirmed      []models.DocumentType
	confirmedErr   error

	hasRequiredCalls []uuid.UUID
	confirmedCalls   []uuid.UUID
}

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

// callCount - общее число вызовов фейка
func (f *fakeDocuments) callCount() int {
	return len(f.hasRequiredCalls) + len(f.confirmedCalls)
}

// testDeps - набор фейков одного кейса
type testDeps struct {
	apps      *fakeApplicationRepository
	sanctions *fakeSanctionsRepository
	customer  *fakeCustomerProfiles
	docs      *fakeDocuments
}

// newTestDeps - свежие фейки в режиме «пустой успех»
func newTestDeps() *testDeps {
	return &testDeps{
		apps:      &fakeApplicationRepository{},
		sanctions: &fakeSanctionsRepository{},
		customer:  &fakeCustomerProfiles{},
		docs:      &fakeDocuments{},
	}
}

// totalCalls - число вызовов всех фейков
func (d *testDeps) totalCalls() int {
	return d.apps.callCount() + d.sanctions.callCount() + d.customer.callCount() + d.docs.callCount()
}

// newTestServer - сервер поверх настоящего сервисного слоя и фейков
func newTestServer(d *testDeps) *KycServer {
	repo := &repository.Repository{Application: d.apps, Sanctions: d.sanctions}
	clients := &client.Clients{Customer: d.customer, Documents: d.docs}
	return NewKycServer(service.NewService(repo, clients), slog.New(slog.DiscardHandler))
}

// requireStatus - ошибка совпадает со статусом домена и имеет ожидаемый код
func requireStatus(t *testing.T, err, wantErr error, wantCode codes.Code) {
	t.Helper()
	require.ErrorIs(t, err, wantErr)
	assert.Equal(t, wantCode, status.Code(err))
}

// assertProtoEqual - сравнение proto-сообщений целиком
func assertProtoEqual(t *testing.T, want, got proto.Message) {
	t.Helper()
	assert.True(t, proto.Equal(want, got), "proto mismatch:\nwant: %v\ngot:  %v", want, got)
}
