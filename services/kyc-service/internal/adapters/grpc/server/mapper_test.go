package server

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

const (
	TEST_USER_ID          = "11111111-1111-1111-1111-111111111111"
	TEST_APPLICATION_ID   = "22222222-2222-2222-2222-222222222222"
	TEST_OFFICER_ID       = "33333333-3333-3333-3333-333333333333"
	TEST_SANCTION_ID      = "44444444-4444-4444-4444-444444444444"
	TEST_CHECK_ID_1       = "55555555-5555-5555-5555-555555555551"
	TEST_CHECK_ID_2       = "55555555-5555-5555-5555-555555555552"
	TEST_EVENT_ID_1       = "66666666-6666-6666-6666-666666666661"
	TEST_EVENT_ID_2       = "66666666-6666-6666-6666-666666666662"
	TEST_APPLICATION_ID_2 = "77777777-7777-7777-7777-777777777777"
)

// testTime - фиксированный момент со смещением в минутах и наносекундами
func testTime(minutes int) time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC).Add(time.Duration(minutes) * time.Minute)
}

// fullApplication - заявка со всеми заполненными полями
func fullApplication() *models.Application {
	applicationID := uuid.MustParse(TEST_APPLICATION_ID)
	officerID := uuid.MustParse(TEST_OFFICER_ID)
	decidedAt := testTime(30)
	birthDate := time.Date(1990, 5, 6, 0, 0, 0, 0, time.UTC)
	return &models.Application{
		ID:             applicationID,
		UserID:         uuid.MustParse(TEST_USER_ID),
		Status:         models.STATUS_DOCS_REQUESTED,
		Reviewer:       &officerID,
		Reason:         "need address",
		RequestedTypes: []models.DocumentType{models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_PROOF_OF_ADDRESS},
		FirstName:      "Ivan",
		LastName:       "Petrov",
		BirthDate:      &birthDate,
		SubmittedAt:    testTime(1),
		DecidedAt:      &decidedAt,
		CreatedAt:      testTime(0),
		UpdatedAt:      testTime(40),
		Checks: []models.Check{
			{ID: uuid.MustParse(TEST_CHECK_ID_1), ApplicationID: applicationID, Type: models.CHECK_TYPE_SANCTIONS, Result: models.CHECK_RESULT_PASSED, Details: "no match", CreatedAt: testTime(2)},
			{ID: uuid.MustParse(TEST_CHECK_ID_2), ApplicationID: applicationID, Type: models.CHECK_TYPE_AGE, Result: models.CHECK_RESULT_FAILED, Details: "under 18", CreatedAt: testTime(3)},
		},
		Events: []models.Event{
			{ID: uuid.MustParse(TEST_EVENT_ID_1), ApplicationID: applicationID, From: "", To: models.STATUS_SUBMITTED, ActorID: nil, Comment: "", CreatedAt: testTime(1)},
			{ID: uuid.MustParse(TEST_EVENT_ID_2), ApplicationID: applicationID, From: models.STATUS_IN_REVIEW, To: models.STATUS_DOCS_REQUESTED, ActorID: &officerID, Comment: "need address", CreatedAt: testTime(30)},
		},
	}
}

// fullApplicationProto - ожидаемый proto для fullApplication, собранный вручную
func fullApplicationProto() *kycv1.Application {
	return &kycv1.Application{
		Id:             TEST_APPLICATION_ID,
		UserId:         TEST_USER_ID,
		Status:         kycv1.ApplicationStatus_APPLICATION_STATUS_DOCS_REQUESTED,
		ReviewerId:     TEST_OFFICER_ID,
		Reason:         "need address",
		RequestedTypes: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_SELFIE, kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT, kycv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS},
		SubmittedAt:    timestamppb.New(testTime(1)),
		DecidedAt:      timestamppb.New(testTime(30)),
		CreatedAt:      timestamppb.New(testTime(0)),
		UpdatedAt:      timestamppb.New(testTime(40)),
		Checks: []*kycv1.KycCheck{
			{Id: TEST_CHECK_ID_1, Type: kycv1.CheckType_CHECK_TYPE_SANCTIONS, Result: kycv1.CheckResult_CHECK_RESULT_PASSED, Details: "no match", CreatedAt: timestamppb.New(testTime(2))},
			{Id: TEST_CHECK_ID_2, Type: kycv1.CheckType_CHECK_TYPE_AGE, Result: kycv1.CheckResult_CHECK_RESULT_FAILED, Details: "under 18", CreatedAt: timestamppb.New(testTime(3))},
		},
		Events: []*kycv1.KycEvent{
			{Id: TEST_EVENT_ID_1, FromStatus: kycv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED, ToStatus: kycv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED, ActorId: "", Comment: "", CreatedAt: timestamppb.New(testTime(1))},
			{Id: TEST_EVENT_ID_2, FromStatus: kycv1.ApplicationStatus_APPLICATION_STATUS_IN_REVIEW, ToStatus: kycv1.ApplicationStatus_APPLICATION_STATUS_DOCS_REQUESTED, ActorId: TEST_OFFICER_ID, Comment: "need address", CreatedAt: timestamppb.New(testTime(30))},
		},
	}
}

// minimalApplication - заявка без рецензента, решения, проверок и событий
func minimalApplication(applicationID string) *models.Application {
	return &models.Application{
		ID:          uuid.MustParse(applicationID),
		UserID:      uuid.MustParse(TEST_USER_ID),
		Status:      models.STATUS_SUBMITTED,
		SubmittedAt: testTime(1),
		CreatedAt:   testTime(0),
		UpdatedAt:   testTime(1),
	}
}

// minimalApplicationProto - ожидаемый proto для minimalApplication
func minimalApplicationProto(applicationID string) *kycv1.Application {
	return &kycv1.Application{
		Id:          applicationID,
		UserId:      TEST_USER_ID,
		Status:      kycv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED,
		SubmittedAt: timestamppb.New(testTime(1)),
		CreatedAt:   timestamppb.New(testTime(0)),
		UpdatedAt:   timestamppb.New(testTime(1)),
	}
}

func TestApplicationToProto(t *testing.T) {
	tests := []struct {
		name string
		app  *models.Application
		want *kycv1.Application
	}{
		{name: "full application", app: fullApplication(), want: fullApplicationProto()},
		{name: "reviewer and decided_at nil, empty checks and events", app: minimalApplication(TEST_APPLICATION_ID), want: minimalApplicationProto(TEST_APPLICATION_ID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applicationToProto(tt.app)

			require.NotNil(t, got)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

func TestApplicationToProto_MinimalFields(t *testing.T) {
	got := applicationToProto(minimalApplication(TEST_APPLICATION_ID))

	require.NotNil(t, got)
	assert.Equal(t, "", got.GetReviewerId())
	assert.Nil(t, got.GetDecidedAt())
	assert.Empty(t, got.GetRequestedTypes())
	assert.Empty(t, got.GetChecks())
	assert.Empty(t, got.GetEvents())
}

// Инвариант: каждый доменный статус переводится в своё proto-значение
func TestApplicationToProto_Status(t *testing.T) {
	tests := []struct {
		name   string
		status models.ApplicationStatus
		want   kycv1.ApplicationStatus
	}{
		{name: "submitted", status: models.STATUS_SUBMITTED, want: kycv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED},
		{name: "auto_checks", status: models.STATUS_AUTO_CHECKS, want: kycv1.ApplicationStatus_APPLICATION_STATUS_AUTO_CHECKS},
		{name: "auto_rejected", status: models.STATUS_AUTO_REJECTED, want: kycv1.ApplicationStatus_APPLICATION_STATUS_AUTO_REJECTED},
		{name: "in_review", status: models.STATUS_IN_REVIEW, want: kycv1.ApplicationStatus_APPLICATION_STATUS_IN_REVIEW},
		{name: "docs_requested", status: models.STATUS_DOCS_REQUESTED, want: kycv1.ApplicationStatus_APPLICATION_STATUS_DOCS_REQUESTED},
		{name: "approved", status: models.STATUS_APPROVED, want: kycv1.ApplicationStatus_APPLICATION_STATUS_APPROVED},
		{name: "rejected", status: models.STATUS_REJECTED, want: kycv1.ApplicationStatus_APPLICATION_STATUS_REJECTED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := minimalApplication(TEST_APPLICATION_ID)
			app.Status = tt.status
			app.Events = []models.Event{{ID: uuid.MustParse(TEST_EVENT_ID_1), From: tt.status, To: tt.status, CreatedAt: testTime(0)}}

			got := applicationToProto(app)

			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.GetStatus())
			require.Len(t, got.GetEvents(), 1)
			assert.Equal(t, tt.want, got.GetEvents()[0].GetFromStatus())
			assert.Equal(t, tt.want, got.GetEvents()[0].GetToStatus())
		})
	}
}

// Инвариант: моменты времени сохраняются до наносекунды
func TestApplicationToProto_Timestamps(t *testing.T) {
	app := fullApplication()

	got := applicationToProto(app)

	require.NotNil(t, got)
	assert.True(t, app.SubmittedAt.Equal(got.GetSubmittedAt().AsTime()))
	assert.True(t, app.CreatedAt.Equal(got.GetCreatedAt().AsTime()))
	assert.True(t, app.UpdatedAt.Equal(got.GetUpdatedAt().AsTime()))
	assert.True(t, app.DecidedAt.Equal(got.GetDecidedAt().AsTime()))
	require.Len(t, got.GetChecks(), 2)
	assert.True(t, app.Checks[1].CreatedAt.Equal(got.GetChecks()[1].GetCreatedAt().AsTime()))
	require.Len(t, got.GetEvents(), 2)
	assert.True(t, app.Events[1].CreatedAt.Equal(got.GetEvents()[1].GetCreatedAt().AsTime()))
}
