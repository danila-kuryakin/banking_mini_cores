package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

var _ interfaces.NotificationRepository = (*fakeNotificationRepository)(nil)
var _ interfaces.Sender = (*fakeSender)(nil)

// saveCall - аргументы одного вызова Save.
type saveCall struct {
	envelope      models.Envelope
	notifications []models.Notification
}

// listCall - аргументы одного вызова ListByUser.
type listCall struct {
	userID uuid.UUID
	page   models.Page
}

// fakeNotificationRepository - in-memory репозиторий с заданными ответами.
type fakeNotificationRepository struct {
	saveResults []bool
	saveErr     error
	saveCalls   []saveCall

	listResult []*models.Notification
	listErr    error
	listCalls  []listCall

	journal *[]string
}

// Save отдаёт ответы из saveResults по очереди, после конца очереди - последний, по умолчанию true.
func (f *fakeNotificationRepository) Save(_ context.Context, envelope models.Envelope, notifications []models.Notification) (bool, error) {
	copied := append([]models.Notification(nil), notifications...)
	f.saveCalls = append(f.saveCalls, saveCall{envelope: envelope, notifications: copied})
	if f.journal != nil {
		*f.journal = append(*f.journal, "save")
	}
	if f.saveErr != nil {
		return false, f.saveErr
	}
	if len(f.saveResults) == 0 {
		return true, nil
	}
	idx := len(f.saveCalls) - 1
	if idx >= len(f.saveResults) {
		idx = len(f.saveResults) - 1
	}
	return f.saveResults[idx], nil
}

func (f *fakeNotificationRepository) ListByUser(_ context.Context, userID uuid.UUID, page models.Page) ([]*models.Notification, error) {
	f.listCalls = append(f.listCalls, listCall{userID: userID, page: page})
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

// fakeSender записывает отправленные уведомления.
type fakeSender struct {
	sendErr   error
	sendCalls []models.Notification
	journal   *[]string
}

func (f *fakeSender) Send(_ context.Context, notification models.Notification) error {
	f.sendCalls = append(f.sendCalls, notification)
	if f.journal != nil {
		*f.journal = append(*f.journal, "send")
	}
	return f.sendErr
}
