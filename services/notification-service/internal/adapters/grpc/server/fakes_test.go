package server

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

var _ interfaces.NotificationRepository = (*fakeNotificationRepository)(nil)

var _ interfaces.Sender = (*fakeSender)(nil)

// listByUserCall - запись одного вызова ListByUser
type listByUserCall struct {
	userID uuid.UUID
	page   models.Page
}

// fakeNotificationRepository - фейк хранилища уведомлений с записью вызовов ListByUser
type fakeNotificationRepository struct {
	mu         sync.Mutex
	listResult []*models.Notification
	listErr    error
	listCalls  []listByUserCall
	saveErr    error
}

func (r *fakeNotificationRepository) Save(_ context.Context, _ models.Envelope, _ []models.Notification) (bool, error) {
	return false, r.saveErr
}

func (r *fakeNotificationRepository) ListByUser(_ context.Context, userID uuid.UUID, page models.Page) ([]*models.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listCalls = append(r.listCalls, listByUserCall{userID: userID, page: page})
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listResult, nil
}

// calls возвращает копию записанных вызовов ListByUser
func (r *fakeNotificationRepository) calls() []listByUserCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listCalls == nil {
		return nil
	}
	return append([]listByUserCall(nil), r.listCalls...)
}

// fakeSender - фейк отправителя, в этом пакете не вызывается
type fakeSender struct {
	sendErr error
}

func (s *fakeSender) Send(_ context.Context, _ models.Notification) error {
	return s.sendErr
}
