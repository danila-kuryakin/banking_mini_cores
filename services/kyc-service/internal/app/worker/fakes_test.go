package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

var (
	_ interfaces.ApplicationRepository = (*fakeApplicationRepository)(nil)
	_ interfaces.CheckRepository       = (*fakeCheckRepository)(nil)
	_ interfaces.SanctionsRepository   = (*fakeSanctionsRepository)(nil)
	_ interfaces.CustomerProfiles      = (*fakeCustomerProfiles)(nil)
	_ interfaces.OutboxRepository      = (*fakeOutboxRepository)(nil)
	_ interfaces.EventPublisher        = (*fakeEventPublisher)(nil)
)

// errNotImplemented - ответ методов фейков, которые юнит не должен вызывать
var errNotImplemented = errors.New("not implemented in fake")

// transitionCall - аргументы одного вызова Transition
type transitionCall struct {
	applicationID uuid.UUID
	transition    models.Transition
	allowedFrom   []models.ApplicationStatus
}

// fakeApplicationRepository - репозиторий заявок с заданной пачкой и ошибками по ID заявки
type fakeApplicationRepository struct {
	mu                sync.Mutex
	takeResult        []*models.Application
	takeErr           error
	takeCalls         []int
	transitionErr     error
	transitionErrByID map[uuid.UUID]error
	transitionCalls   []transitionCall
}

func (f *fakeApplicationRepository) Create(_ context.Context, _ models.Application) (*models.Application, error) {
	return nil, errNotImplemented
}

func (f *fakeApplicationRepository) GetByID(_ context.Context, _ uuid.UUID) (*models.Application, error) {
	return nil, errNotImplemented
}

func (f *fakeApplicationRepository) GetLatestByUserID(_ context.Context, _ uuid.UUID) (*models.Application, error) {
	return nil, errNotImplemented
}

func (f *fakeApplicationRepository) List(_ context.Context, _ models.ApplicationFilter) ([]*models.Application, error) {
	return nil, errNotImplemented
}

func (f *fakeApplicationRepository) Transition(_ context.Context, applicationID uuid.UUID, t models.Transition, allowedFrom []models.ApplicationStatus) (*models.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transitionCalls = append(f.transitionCalls, transitionCall{applicationID: applicationID, transition: t, allowedFrom: allowedFrom})
	if err, ok := f.transitionErrByID[applicationID]; ok {
		return nil, err
	}
	if f.transitionErr != nil {
		return nil, f.transitionErr
	}
	return &models.Application{ID: applicationID}, nil
}

func (f *fakeApplicationRepository) TakeForAutoChecks(_ context.Context, limit int) ([]*models.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.takeCalls = append(f.takeCalls, limit)
	if f.takeErr != nil {
		return nil, f.takeErr
	}
	return f.takeResult, nil
}

func (f *fakeApplicationRepository) Claim(_ context.Context, _, _ uuid.UUID) (*models.Application, error) {
	return nil, errNotImplemented
}

// takes возвращает копию записанных лимитов TakeForAutoChecks
func (f *fakeApplicationRepository) takes() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.takeCalls...)
}

// transitions возвращает копию записанных вызовов Transition
func (f *fakeApplicationRepository) transitions() []transitionCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]transitionCall(nil), f.transitionCalls...)
}

// fakeCheckRepository - хранилище результатов проверок с ошибкой на заданном по счёту вызове
type fakeCheckRepository struct {
	mu            sync.Mutex
	saveErr       error
	saveErrOnCall int
	saveCalls     []models.Check
}

func (f *fakeCheckRepository) SaveCheck(_ context.Context, check models.Check) (*models.Check, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saveCalls = append(f.saveCalls, check)
	if f.saveErr != nil && len(f.saveCalls) == f.saveErrOnCall {
		return nil, f.saveErr
	}
	saved := check
	return &saved, nil
}

// saves возвращает копию записанных вызовов SaveCheck
func (f *fakeCheckRepository) saves() []models.Check {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]models.Check(nil), f.saveCalls...)
}

// fakeSanctionsRepository - санкционный список с одной заданной записью-совпадением
type fakeSanctionsRepository struct {
	mu           sync.Mutex
	match        *models.SanctionEntry
	findMatchErr error
}

func (f *fakeSanctionsRepository) FindMatch(_ context.Context, _, _ string, _ time.Time) (*models.SanctionEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.findMatchErr != nil {
		return nil, f.findMatchErr
	}
	if f.match == nil {
		return nil, nil
	}
	entry := *f.match
	return &entry, nil
}

func (f *fakeSanctionsRepository) Add(_ context.Context, _ models.SanctionEntry) (*models.SanctionEntry, error) {
	return nil, errNotImplemented
}

func (f *fakeSanctionsRepository) List(_ context.Context, _, _ int) ([]models.SanctionEntry, error) {
	return nil, errNotImplemented
}

// setStatusCall - аргументы одного вызова SetStatus
type setStatusCall struct {
	userID  uuid.UUID
	to      models.CustomerStatus
	reason  string
	actorID *uuid.UUID
}

// fakeCustomerProfiles - customer-service с ошибкой SetStatus общей или по пользователю
type fakeCustomerProfiles struct {
	mu                 sync.Mutex
	setStatusErr       error
	setStatusErrByUser map[uuid.UUID]error
	setStatusCalls     []setStatusCall
}

func (f *fakeCustomerProfiles) GetProfile(_ context.Context, _ uuid.UUID) (*models.CustomerProfile, error) {
	return nil, errNotImplemented
}

func (f *fakeCustomerProfiles) SetStatus(_ context.Context, userID uuid.UUID, to models.CustomerStatus, reason string, actorID *uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setStatusCalls = append(f.setStatusCalls, setStatusCall{userID: userID, to: to, reason: reason, actorID: actorID})
	if err, ok := f.setStatusErrByUser[userID]; ok {
		return err
	}
	return f.setStatusErr
}

// setStatuses возвращает копию записанных вызовов SetStatus
func (f *fakeCustomerProfiles) setStatuses() []setStatusCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]setStatusCall(nil), f.setStatusCalls...)
}

// relayResponse - заданный ответ одного вызова Relay
type relayResponse struct {
	sent int
	err  error
	msgs []models.OutboxMessage
}

// fakeOutboxRepository - outbox с очередью ответов; пустая очередь даёт (0, nil)
type fakeOutboxRepository struct {
	mu         sync.Mutex
	responses  []relayResponse
	relayCalls []int
}

func (f *fakeOutboxRepository) Relay(ctx context.Context, limit int, publish func(ctx context.Context, msgs []models.OutboxMessage) error) (int, error) {
	f.mu.Lock()
	f.relayCalls = append(f.relayCalls, limit)
	if len(f.responses) == 0 {
		f.mu.Unlock()
		return 0, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	f.mu.Unlock()
	if resp.err != nil {
		return 0, resp.err
	}
	if resp.msgs != nil {
		if err := publish(ctx, resp.msgs); err != nil {
			return 0, err
		}
	}
	return resp.sent, nil
}

// relays возвращает копию записанных лимитов Relay
func (f *fakeOutboxRepository) relays() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.relayCalls...)
}

// fakeEventPublisher - издатель событий с записью отправленных пачек
type fakeEventPublisher struct {
	mu           sync.Mutex
	publishErr   error
	publishCalls [][]models.OutboxMessage
}

func (f *fakeEventPublisher) Publish(_ context.Context, msgs []models.OutboxMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.publishCalls = append(f.publishCalls, msgs)
	return f.publishErr
}

// publishes возвращает копию записанных пачек Publish
func (f *fakeEventPublisher) publishes() [][]models.OutboxMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]models.OutboxMessage(nil), f.publishCalls...)
}
