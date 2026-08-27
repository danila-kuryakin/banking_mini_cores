package kafka

import (
	"strings"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
)

// Типы событий, на которые сервис реагирует осмысленно. Всё остальное тоже
// сохраняется - просто с каналом по умолчанию и темой из самого типа: пусть
// незнакомое событие видно в ListNotifications, а не теряется молча.
const (
	EventUserRegistered = "auth.user.registered"

	EventProfileUpdated = "customer.profile.updated"
	EventStatusChanged  = "customer.status.changed"

	EventApplicationSubmitted = "kyc.application.submitted"
	EventApplicationApproved  = "kyc.application.approved"
	EventApplicationRejected  = "kyc.application.rejected"

	EventTransactionPosted = "ledger.transaction.posted"
	EventTransactionFailed = "ledger.transaction.failed"
)

// subjects — тема письма для известных событий.
var subjects = map[string]string{
	EventUserRegistered: "Welcome aboard",

	EventProfileUpdated: "Your profile has been updated",
	EventStatusChanged:  "Your account status has changed",

	EventApplicationSubmitted: "We received your application",
	EventApplicationApproved:  "Your application is approved",
	EventApplicationRejected:  "Your application was declined",

	EventTransactionPosted: "Transaction completed",
	EventTransactionFailed: "Transaction declined",
}

// channels — канал доставки для известных событий.
//
// Деньги и отказы идут в SMS: их читают сразу, даже когда почта ждёт до
// вечера. Остальное - почтой.
var channels = map[string]notification.Channel{
	EventUserRegistered: notification.ChannelEmail,

	EventProfileUpdated: notification.ChannelEmail,
	EventStatusChanged:  notification.ChannelEmail,

	EventApplicationSubmitted: notification.ChannelEmail,
	EventApplicationApproved:  notification.ChannelEmail,
	EventApplicationRejected:  notification.ChannelSMS,

	EventTransactionPosted: notification.ChannelSMS,
	EventTransactionFailed: notification.ChannelSMS,
}

// subjectFor подбирает тему по типу события.
//
// Для незнакомого типа темой становится его последняя часть: из
// "billing.invoice.issued" выйдет "Invoice issued". Читается это, конечно,
// хуже написанной вручную темы, но сервис при этом остаётся полезным, когда
// в шине появляется новое событие, а сюда его добавить ещё не успели.
func subjectFor(eventType string) string {
	if s, ok := subjects[eventType]; ok {
		return s
	}

	parts := strings.Split(eventType, ".")
	if len(parts) < 2 {
		return eventType
	}

	// Тип вида "billing." даёт пустой хвост, и обращение к tail[:1] уронило бы
	// консьюмера на разборе одного кривого сообщения.
	tail := strings.TrimSpace(strings.Join(parts[1:], " "))
	if tail == "" {
		return eventType
	}

	return strings.ToUpper(tail[:1]) + tail[1:]
}

// channelFor подбирает канал по типу события.
func channelFor(eventType string) notification.Channel {
	if c, ok := channels[eventType]; ok {
		return c
	}

	return notification.ChannelEmail
}
