package templates

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

const (
	TEST_SENDER    = "11111111-1111-4111-8111-111111111111"
	TEST_RECIPIENT = "22222222-2222-4222-8222-222222222222"
)

func envelope(t *testing.T, eventType string, payload any) models.Envelope {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return models.Envelope{EventID: uuid.New(), EventType: eventType, Payload: raw}
}

func recipients(notifications []models.Notification) []string {
	out := make([]string, 0, len(notifications))
	for _, n := range notifications {
		out = append(out, n.UserID.String())
	}

	return out
}

func TestTransferCompletedNotifiesBothSides(t *testing.T) {
	env := envelope(t, models.EVENT_TRANSACTION_COMPLETED, models.TransactionPayload{
		Type:       models.TRANSACTION_TYPE_TRANSFER,
		Amount:     150050,
		Currency:   "RUB",
		FromUserID: TEST_SENDER,
		ToUserID:   TEST_RECIPIENT,
	})

	got, err := Build(env)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if strings.Join(recipients(got), ",") != TEST_SENDER+","+TEST_RECIPIENT {
		t.Fatalf("recipients = %v", recipients(got))
	}

	if !strings.Contains(got[0].Text, "1500.50 RUB") {
		t.Fatalf("amount is not formatted: %q", got[0].Text)
	}

	if got[0].EventID != env.EventID || got[0].EventType != env.EventType {
		t.Fatalf("event is not carried into notification")
	}
}

func TestTransferBetweenOwnAccountsNotifiesOnce(t *testing.T) {
	got, err := Build(envelope(t, models.EVENT_TRANSACTION_COMPLETED, models.TransactionPayload{
		Type:       models.TRANSACTION_TYPE_TRANSFER,
		Amount:     100,
		Currency:   "RUB",
		FromUserID: TEST_SENDER,
		ToUserID:   TEST_SENDER,
	}))
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("want one notification, got %d", len(got))
	}
}

func TestDepositNotifiesRecipient(t *testing.T) {
	got, err := Build(envelope(t, models.EVENT_TRANSACTION_COMPLETED, models.TransactionPayload{
		Type:        models.TRANSACTION_TYPE_DEPOSIT,
		Amount:      100,
		Currency:    "USD",
		InitiatorID: TEST_SENDER,
		ToUserID:    TEST_RECIPIENT,
	}))
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if strings.Join(recipients(got), ",") != TEST_RECIPIENT {
		t.Fatalf("deposit must notify only the recipient, got %v", recipients(got))
	}
}

func TestDeclinedNotifiesSenderOnly(t *testing.T) {
	got, err := Build(envelope(t, models.EVENT_TRANSACTION_DECLINED, models.TransactionPayload{
		Type:          models.TRANSACTION_TYPE_TRANSFER,
		Amount:        100,
		Currency:      "RUB",
		InitiatorID:   TEST_SENDER,
		ToUserID:      TEST_RECIPIENT,
		DeclineReason: "insufficient funds",
	}))
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if strings.Join(recipients(got), ",") != TEST_SENDER {
		t.Fatalf("declined must notify only the sender, got %v", recipients(got))
	}

	if !strings.Contains(got[0].Text, "insufficient funds") {
		t.Fatalf("reason is missing: %q", got[0].Text)
	}
}

func TestKycEvents(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		payload   models.ApplicationPayload
		contains  string
	}{
		{"approved", models.EVENT_APPLICATION_APPROVED, models.ApplicationPayload{UserID: TEST_SENDER, Status: "approved"}, "верифицированы"},
		{"rejected by officer", models.EVENT_APPLICATION_REJECTED, models.ApplicationPayload{UserID: TEST_SENDER, Status: "rejected", Reason: "fake passport"}, "отклонена: fake passport"},
		{"auto rejected", models.EVENT_APPLICATION_REJECTED, models.ApplicationPayload{UserID: TEST_SENDER, Status: "auto_rejected", Reason: "age 16 is under 18"}, "подайте заявку снова"},
		{"docs requested", models.EVENT_DOCS_REQUESTED, models.ApplicationPayload{UserID: TEST_SENDER, Status: "docs_requested", RequestedTypes: []string{"passport", "proof_of_address"}}, "паспорт, подтверждение адреса"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Build(envelope(t, tc.eventType, tc.payload))
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			if len(got) != 1 || got[0].UserID.String() != TEST_SENDER {
				t.Fatalf("want one notification to the client, got %v", recipients(got))
			}

			if !strings.Contains(got[0].Text, tc.contains) {
				t.Fatalf("text %q does not contain %q", got[0].Text, tc.contains)
			}
		})
	}
}

func TestSubmittedIsSkipped(t *testing.T) {
	got, err := Build(envelope(t, models.EVENT_APPLICATION_SUBMITTED, models.ApplicationPayload{UserID: TEST_SENDER}))
	if err != nil || len(got) != 0 {
		t.Fatalf("submitted must not notify, got %v, %v", got, err)
	}
}

func TestUnknownEventIsSkipped(t *testing.T) {
	got, err := Build(models.Envelope{EventID: uuid.New(), EventType: "ledger.something_new", Payload: []byte(`{}`)})
	if err != nil || len(got) != 0 {
		t.Fatalf("unknown event must be skipped, got %v, %v", got, err)
	}
}

func TestPoisonEvents(t *testing.T) {
	cases := []struct {
		name string
		env  models.Envelope
	}{
		{"broken payload", models.Envelope{EventID: uuid.New(), EventType: models.EVENT_TRANSACTION_COMPLETED, Payload: []byte(`{"amount":"x"}`)}},
		{"unknown transaction type", envelope(t, models.EVENT_TRANSACTION_COMPLETED, models.TransactionPayload{Type: "refund"})},
		{"bad user id", envelope(t, models.EVENT_TRANSACTION_DECLINED, models.TransactionPayload{FromUserID: "not-a-uuid"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Build(tc.env); !errors.Is(err, domain.ErrPoisonMessage) {
				t.Fatalf("want poison error, got %v", err)
			}
		})
	}
}
