package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
)

func TestSubjectFor(t *testing.T) {
	cases := map[string]string{
		EventUserRegistered:      "Welcome aboard",
		EventTransactionFailed:   "Transaction declined",
		"billing.invoice.issued": "Invoice issued", // незнакомый тип
		"billing.":               "billing.",       // пустой хвост не должен ронять
		"weird":                  "weird",
		"":                       "",
	}

	for in, want := range cases {
		if got := subjectFor(in); got != want {
			t.Errorf("subjectFor(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

func TestChannelForUnknownEventFallsBackToEmail(t *testing.T) {
	if got := channelFor("billing.invoice.issued"); got != notification.ChannelEmail {
		t.Fatalf("channelFor(неизвестный) = %q, ожидался email", got)
	}

	if got := channelFor(EventTransactionPosted); got != notification.ChannelSMS {
		t.Fatalf("channelFor(transaction.posted) = %q, ожидался sms", got)
	}
}

func TestBuildUsesUserIDWhenCustomerIsMissing(t *testing.T) {
	occurred := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

	env := Envelope{
		ID:         "evt-1",
		Type:       EventUserRegistered,
		Source:     "auth-service",
		OccurredAt: occurred,
		Payload:    json.RawMessage(`{"user_id":"u-1","email":"a@b.c"}`),
	}

	var p payload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatal(err)
	}

	n := build(env, p)

	// При регистрации клиента ещё нет, адресатом остаётся пользователь.
	if n.CustomerID != "u-1" {
		t.Fatalf("CustomerID = %q, ожидался %q", n.CustomerID, "u-1")
	}
	if n.Status != notification.StatusPending {
		t.Fatalf("Status = %q, ожидался pending", n.Status)
	}
	if !n.CreatedAt.Equal(occurred) {
		t.Fatalf("CreatedAt = %v, ожидалось время события %v", n.CreatedAt, occurred)
	}
	if n.EventID != "evt-1" {
		t.Fatalf("EventID = %q", n.EventID)
	}
}

func TestBuildFallsBackToNowWhenTimeIsMissing(t *testing.T) {
	n := build(Envelope{ID: "evt-2", Type: "x.y.z"}, payload{})

	if n.CreatedAt.IsZero() {
		t.Fatal("CreatedAt пустое: событие без occurred_at должно получить текущее время")
	}
}
