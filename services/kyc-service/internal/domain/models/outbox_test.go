package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_APPLICATION_ID = "0195a1b2-0000-7000-8000-000000000001"
	TEST_USER_ID        = "0195a1b2-0000-7000-8000-000000000002"
	TEST_OFFICER_ID     = "0195a1b2-0000-7000-8000-000000000003"
	TEST_REASON         = "need address"
)

// testUpdatedAt - 2026-03-01 15:00 в зоне UTC+3, то есть 12:00 UTC.
func testUpdatedAt() time.Time {
	return time.Date(2026, time.March, 1, 15, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
}

// testUpdatedAtUTC - тот же момент, что testUpdatedAt, в UTC.
func testUpdatedAtUTC() time.Time {
	return time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
}

func testApplication(status ApplicationStatus, reason string, requestedTypes []DocumentType) *Application {
	return &Application{
		ID:             uuid.MustParse(TEST_APPLICATION_ID),
		UserID:         uuid.MustParse(TEST_USER_ID),
		Status:         status,
		Reason:         reason,
		RequestedTypes: requestedTypes,
		UpdatedAt:      testUpdatedAt(),
	}
}

func TestNewApplicationEvent_Published(t *testing.T) {
	officerID := uuid.MustParse(TEST_OFFICER_ID)

	tests := []struct {
		name          string
		app           *Application
		actorID       *uuid.UUID
		wantEventType string
		wantEvent     ApplicationEvent
		wantKeys      []string
		wantNoKeys    []string
	}{
		{
			name:          "submitted",
			app:           testApplication(STATUS_SUBMITTED, "", nil),
			actorID:       nil,
			wantEventType: EVENT_APPLICATION_SUBMITTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "submitted", UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"application_id", "user_id", "status", "updated_at"},
			wantNoKeys: []string{"reason", "requested_types", "actor_id"},
		},
		{
			name:          "approved_without_reason_and_actor",
			app:           testApplication(STATUS_APPROVED, "", nil),
			actorID:       nil,
			wantEventType: EVENT_APPLICATION_APPROVED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "approved", UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"application_id", "user_id", "status", "updated_at"},
			wantNoKeys: []string{"reason", "requested_types", "actor_id"},
		},
		{
			name:          "rejected_by_officer",
			app:           testApplication(STATUS_REJECTED, "fake passport", nil),
			actorID:       &officerID,
			wantEventType: EVENT_APPLICATION_REJECTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "rejected", Reason: "fake passport",
				ActorID: TEST_OFFICER_ID, UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"reason", "actor_id"},
			wantNoKeys: []string{"requested_types"},
		},
		{
			name:          "auto_rejected_by_system",
			app:           testApplication(STATUS_AUTO_REJECTED, "sanctions match", nil),
			actorID:       nil,
			wantEventType: EVENT_APPLICATION_REJECTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "auto_rejected", Reason: "sanctions match",
				UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"reason"},
			wantNoKeys: []string{"requested_types", "actor_id"},
		},
		{
			name:          "docs_requested_full_example",
			app:           testApplication(STATUS_DOCS_REQUESTED, TEST_REASON, []DocumentType{DOCUMENT_TYPE_PROOF_OF_ADDRESS, DOCUMENT_TYPE_PASSPORT}),
			actorID:       &officerID,
			wantEventType: EVENT_DOCS_REQUESTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "docs_requested", Reason: TEST_REASON,
				RequestedTypes: []string{"proof_of_address", "passport"}, ActorID: TEST_OFFICER_ID, UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"application_id", "user_id", "status", "reason", "requested_types", "actor_id", "updated_at"},
			wantNoKeys: []string{},
		},
		{
			name:          "docs_requested_nil_requested_types",
			app:           testApplication(STATUS_DOCS_REQUESTED, TEST_REASON, nil),
			actorID:       &officerID,
			wantEventType: EVENT_DOCS_REQUESTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "docs_requested", Reason: TEST_REASON,
				ActorID: TEST_OFFICER_ID, UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"reason", "actor_id"},
			wantNoKeys: []string{"requested_types"},
		},
		{
			name:          "docs_requested_empty_requested_types",
			app:           testApplication(STATUS_DOCS_REQUESTED, TEST_REASON, []DocumentType{}),
			actorID:       &officerID,
			wantEventType: EVENT_DOCS_REQUESTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "docs_requested", Reason: TEST_REASON,
				ActorID: TEST_OFFICER_ID, UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"reason", "actor_id"},
			wantNoKeys: []string{"requested_types"},
		},
		{
			name:          "approved_ignores_requested_types",
			app:           testApplication(STATUS_APPROVED, "", []DocumentType{DOCUMENT_TYPE_PASSPORT}),
			actorID:       &officerID,
			wantEventType: EVENT_APPLICATION_APPROVED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "approved",
				ActorID: TEST_OFFICER_ID, UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"actor_id"},
			wantNoKeys: []string{"requested_types", "reason"},
		},
		{
			name:          "rejected_ignores_requested_types",
			app:           testApplication(STATUS_REJECTED, "bad selfie", []DocumentType{DOCUMENT_TYPE_SELFIE}),
			actorID:       &officerID,
			wantEventType: EVENT_APPLICATION_REJECTED,
			wantEvent: ApplicationEvent{
				ApplicationID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: "rejected", Reason: "bad selfie",
				ActorID: TEST_OFFICER_ID, UpdatedAt: testUpdatedAtUTC(),
			},
			wantKeys:   []string{"reason", "actor_id"},
			wantNoKeys: []string{"requested_types"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, ok, err := NewApplicationEvent(tt.app, tt.actorID)
			require.NoError(t, err)
			require.True(t, ok)

			assert.Equal(t, tt.wantEventType, msg.EventType)
			assert.Equal(t, TEST_USER_ID, msg.Key)
			assert.Equal(t, time.UTC, msg.OccurredAt.Location())
			assert.True(t, testUpdatedAtUTC().Equal(msg.OccurredAt), "occurred_at %v", msg.OccurredAt)

			var gotEvent ApplicationEvent
			require.NoError(t, json.Unmarshal(msg.Payload, &gotEvent))
			assert.Equal(t, tt.wantEvent, gotEvent)

			var gotRaw map[string]any
			require.NoError(t, json.Unmarshal(msg.Payload, &gotRaw))
			for _, key := range tt.wantKeys {
				assert.Contains(t, gotRaw, key)
			}
			for _, key := range tt.wantNoKeys {
				assert.NotContains(t, gotRaw, key)
			}
			assert.Equal(t, "2026-03-01T12:00:00Z", gotRaw["updated_at"])
		})
	}
}

func TestNewApplicationEvent_NotPublished(t *testing.T) {
	officerID := uuid.MustParse(TEST_OFFICER_ID)

	tests := []struct {
		name   string
		status ApplicationStatus
	}{
		{name: "auto_checks", status: STATUS_AUTO_CHECKS},
		{name: "in_review", status: STATUS_IN_REVIEW},
		{name: "empty", status: ""},
		{name: "draft", status: "draft"},
		{name: "upper_case", status: "SUBMITTED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testApplication(tt.status, TEST_REASON, []DocumentType{DOCUMENT_TYPE_PASSPORT})

			msg, ok, err := NewApplicationEvent(app, &officerID)
			require.NoError(t, err)
			assert.False(t, ok)
			assert.Equal(t, OutboxMessage{}, msg)
		})
	}
}

func TestNewApplicationEvent_OccurredAtUTC(t *testing.T) {
	tests := []struct {
		name      string
		updatedAt time.Time
	}{
		{name: "non_utc_zone", updatedAt: testUpdatedAt()},
		{name: "already_utc", updatedAt: testUpdatedAtUTC()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testApplication(STATUS_SUBMITTED, "", nil)
			app.UpdatedAt = tt.updatedAt

			msg, ok, err := NewApplicationEvent(app, nil)
			require.NoError(t, err)
			require.True(t, ok)

			assert.Equal(t, time.UTC, msg.OccurredAt.Location())
			assert.True(t, tt.updatedAt.Equal(msg.OccurredAt), "occurred_at %v", msg.OccurredAt)
			assert.Equal(t, testUpdatedAtUTC(), msg.OccurredAt)
		})
	}
}

func TestNewApplicationEvent_UniqueV7IDs(t *testing.T) {
	app := testApplication(STATUS_SUBMITTED, "", nil)
	seen := make(map[uuid.UUID]struct{})

	for range 3 {
		msg, ok, err := NewApplicationEvent(app, nil)
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, uuid.Version(7), msg.ID.Version())
		assert.NotContains(t, seen, msg.ID)
		seen[msg.ID] = struct{}{}
	}
	assert.Len(t, seen, 3)
}

func TestNewApplicationEvent_KeyIsUserID(t *testing.T) {
	tests := []struct {
		name   string
		status ApplicationStatus
	}{
		{name: "submitted", status: STATUS_SUBMITTED},
		{name: "approved", status: STATUS_APPROVED},
		{name: "docs_requested", status: STATUS_DOCS_REQUESTED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testApplication(tt.status, "", nil)

			msg, ok, err := NewApplicationEvent(app, nil)
			require.NoError(t, err)
			require.True(t, ok)

			assert.Equal(t, TEST_USER_ID, msg.Key)
			assert.NotEqual(t, TEST_APPLICATION_ID, msg.Key)
		})
	}
}
