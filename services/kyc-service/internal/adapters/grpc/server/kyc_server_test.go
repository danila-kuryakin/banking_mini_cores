package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

const (
	TEST_BAD_UUID = "not-a-uuid"
	TEST_COMMENT  = "looks good"
	TEST_REASON   = "fake documents"
)

// filledProfile - анкета, проходящая предусловия подачи
func filledProfile() *models.CustomerProfile {
	birthDate := time.Date(1990, 5, 6, 0, 0, 0, 0, time.UTC)
	return &models.CustomerProfile{FirstName: "Ivan", LastName: "Petrov", BirthDate: &birthDate, Status: models.CUSTOMER_STATUS_PROFILE_FILLED}
}

// readyForSubmit - фейки, при которых подача доходит до Create
func readyForSubmit(d *testDeps) {
	d.apps.latestErr = domain.ErrApplicationNotFound
	d.customer.profileResult = filledProfile()
	d.docs.hasRequired = true
}

// R1: разбор идентификаторов; при ошибке зависимости не вызываются
func TestKycServer_ParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		call    func(ctx context.Context, srv *KycServer) error
		wantErr error
	}{
		{name: "submit invalid user_id", wantErr: domain.ErrUserIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.SubmitApplication(ctx, &kycv1.SubmitApplicationRequest{UserId: TEST_BAD_UUID})
			return err
		}},
		{name: "submit empty user_id", wantErr: domain.ErrUserIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.SubmitApplication(ctx, &kycv1.SubmitApplicationRequest{UserId: ""})
			return err
		}},
		{name: "get by id invalid application_id", wantErr: domain.ErrApplicationIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.GetApplicationByID(ctx, &kycv1.GetApplicationByIDRequest{ApplicationId: TEST_BAD_UUID})
			return err
		}},
		{name: "get by id empty application_id", wantErr: domain.ErrApplicationIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.GetApplicationByID(ctx, &kycv1.GetApplicationByIDRequest{ApplicationId: ""})
			return err
		}},
		{name: "get by user id invalid user_id", wantErr: domain.ErrUserIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.GetApplicationByUserID(ctx, &kycv1.GetApplicationByUserIDRequest{UserId: TEST_BAD_UUID})
			return err
		}},
		{name: "get by user id empty user_id", wantErr: domain.ErrUserIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.GetApplicationByUserID(ctx, &kycv1.GetApplicationByUserIDRequest{UserId: ""})
			return err
		}},
		{name: "list invalid reviewer_id", wantErr: domain.ErrOfficerIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.ListApplications(ctx, &kycv1.ListApplicationsRequest{ReviewerId: TEST_BAD_UUID})
			return err
		}},
		{name: "claim both invalid - application_id first", wantErr: domain.ErrApplicationIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.ClaimApplication(ctx, &kycv1.ClaimApplicationRequest{ApplicationId: TEST_BAD_UUID, OfficerId: TEST_BAD_UUID})
			return err
		}},
		{name: "claim empty officer_id", wantErr: domain.ErrOfficerIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.ClaimApplication(ctx, &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: ""})
			return err
		}},
		{name: "approve both invalid - application_id first", wantErr: domain.ErrApplicationIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.ApproveApplication(ctx, &kycv1.ApproveApplicationRequest{ApplicationId: TEST_BAD_UUID, OfficerId: TEST_BAD_UUID})
			return err
		}},
		{name: "approve invalid officer_id", wantErr: domain.ErrOfficerIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.ApproveApplication(ctx, &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BAD_UUID})
			return err
		}},
		{name: "reject both invalid - application_id first", wantErr: domain.ErrApplicationIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.RejectApplication(ctx, &kycv1.RejectApplicationRequest{ApplicationId: TEST_BAD_UUID, OfficerId: TEST_BAD_UUID, Reason: TEST_REASON})
			return err
		}},
		{name: "reject invalid officer_id", wantErr: domain.ErrOfficerIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.RejectApplication(ctx, &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BAD_UUID, Reason: TEST_REASON})
			return err
		}},
		{name: "request documents both invalid - application_id first", wantErr: domain.ErrApplicationIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.RequestMoreDocuments(ctx, &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_BAD_UUID, OfficerId: TEST_BAD_UUID, Types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT}})
			return err
		}},
		{name: "request documents invalid officer_id", wantErr: domain.ErrOfficerIDMustBeUUID, call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.RequestMoreDocuments(ctx, &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BAD_UUID, Types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT}})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			srv := newTestServer(d)

			err := tt.call(t.Context(), srv)

			requireStatus(t, err, tt.wantErr, codes.InvalidArgument)
			assert.Equal(t, 0, d.totalCalls())
		})
	}
}

// Инвариант R4: неразобранная ошибка - Internal-заглушка своего метода
func TestKycServer_InternalErrors(t *testing.T) {
	dbErr := errors.New("db down")
	tests := []struct {
		name    string
		setup   func(d *testDeps)
		call    func(ctx context.Context, srv *KycServer) error
		wantErr error
	}{
		{name: "submit", wantErr: domain.ErrSubmitFailed,
			setup: func(d *testDeps) { d.apps.latestErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.SubmitApplication(ctx, &kycv1.SubmitApplicationRequest{UserId: TEST_USER_ID})
				return err
			}},
		{name: "get by id", wantErr: domain.ErrGetFailed,
			setup: func(d *testDeps) { d.apps.getByIDErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.GetApplicationByID(ctx, &kycv1.GetApplicationByIDRequest{ApplicationId: TEST_APPLICATION_ID})
				return err
			}},
		{name: "get by user id", wantErr: domain.ErrGetFailed,
			setup: func(d *testDeps) { d.apps.latestErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.GetApplicationByUserID(ctx, &kycv1.GetApplicationByUserIDRequest{UserId: TEST_USER_ID})
				return err
			}},
		{name: "list applications", wantErr: domain.ErrListFailed,
			setup: func(d *testDeps) { d.apps.listErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.ListApplications(ctx, &kycv1.ListApplicationsRequest{})
				return err
			}},
		{name: "claim", wantErr: domain.ErrReviewFailed,
			setup: func(d *testDeps) { d.apps.claimErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.ClaimApplication(ctx, &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID})
				return err
			}},
		{name: "approve", wantErr: domain.ErrReviewFailed,
			setup: func(d *testDeps) { d.apps.transitionErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.ApproveApplication(ctx, &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT})
				return err
			}},
		{name: "reject", wantErr: domain.ErrReviewFailed,
			setup: func(d *testDeps) { d.apps.transitionErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.RejectApplication(ctx, &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Reason: TEST_REASON})
				return err
			}},
		{name: "reject set status fails", wantErr: domain.ErrReviewFailed,
			setup: func(d *testDeps) {
				d.apps.transitionResult = minimalApplication(TEST_APPLICATION_ID)
				d.customer.setStatusErr = dbErr
			},
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.RejectApplication(ctx, &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Reason: TEST_REASON})
				return err
			}},
		{name: "request more documents", wantErr: domain.ErrReviewFailed,
			setup: func(d *testDeps) { d.apps.transitionErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.RequestMoreDocuments(ctx, &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT}})
				return err
			}},
		{name: "add sanction entry", wantErr: domain.ErrAddSanctionFailed,
			setup: func(d *testDeps) { d.sanctions.addErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.AddSanctionEntry(ctx, &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1980, Month: 1, Day: 15}, Source: "OFAC"})
				return err
			}},
		{name: "list sanctions", wantErr: domain.ErrListSanctionsFailed,
			setup: func(d *testDeps) { d.sanctions.listErr = dbErr },
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.ListSanctions(ctx, &kycv1.ListSanctionsRequest{})
				return err
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			err := tt.call(t.Context(), srv)

			requireStatus(t, err, tt.wantErr, codes.Internal)
		})
	}
}

// R3: статус соседнего сервиса уходит наружу со своим кодом
func TestKycServer_UpstreamStatusPassThrough(t *testing.T) {
	unavailable := status.Error(codes.Unavailable, "customer-service is down")
	tests := []struct {
		name  string
		setup func(d *testDeps)
		call  func(ctx context.Context, srv *KycServer) error
	}{
		{name: "submit get profile bare status",
			setup: func(d *testDeps) {
				d.apps.latestErr = domain.ErrApplicationNotFound
				d.customer.getProfileErr = unavailable
			},
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.SubmitApplication(ctx, &kycv1.SubmitApplicationRequest{UserId: TEST_USER_ID})
				return err
			}},
		{name: "submit get profile wrapped status",
			setup: func(d *testDeps) {
				d.apps.latestErr = domain.ErrApplicationNotFound
				d.customer.getProfileErr = fmt.Errorf("get profile: %w", unavailable)
			},
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.SubmitApplication(ctx, &kycv1.SubmitApplicationRequest{UserId: TEST_USER_ID})
				return err
			}},
		{name: "approve set status",
			setup: func(d *testDeps) {
				d.apps.transitionResult = minimalApplication(TEST_APPLICATION_ID)
				d.customer.setStatusErr = unavailable
			},
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.ApproveApplication(ctx, &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT})
				return err
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			err := tt.call(t.Context(), srv)

			require.Error(t, err)
			assert.Equal(t, codes.Unavailable, status.Code(err))
		})
	}
}

func TestKycServer_SubmitApplication(t *testing.T) {
	inReview := minimalApplication(TEST_APPLICATION_ID)
	inReview.Status = models.STATUS_IN_REVIEW
	tests := []struct {
		name     string
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "success returns created application",
			setup: func(d *testDeps) {
				readyForSubmit(d)
				d.apps.createResult = minimalApplication(TEST_APPLICATION_ID)
			},
			want: minimalApplicationProto(TEST_APPLICATION_ID)},
		{name: "active application exists",
			setup:   func(d *testDeps) { d.apps.latestResult = inReview },
			wantErr: domain.ErrApplicationExists, wantCode: codes.AlreadyExists},
		{name: "profile not filled",
			setup: func(d *testDeps) {
				readyForSubmit(d)
				d.customer.profileResult = &models.CustomerProfile{Status: models.CUSTOMER_STATUS_NEW}
			},
			wantErr: domain.ErrProfileIsNotFilled, wantCode: codes.FailedPrecondition},
		{name: "customer not found on set status (wrapped)",
			setup: func(d *testDeps) {
				readyForSubmit(d)
				d.customer.setStatusErr = domain.ErrCustomerNotFound
			},
			wantErr: domain.ErrCustomerMissing, wantCode: codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.SubmitApplication(t.Context(), &kycv1.SubmitApplicationRequest{UserId: TEST_USER_ID})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

// R2: нехватка документов - FailedPrecondition с полным текстом ошибки
func TestKycServer_SubmitApplication_DocumentsMissing(t *testing.T) {
	d := newTestDeps()
	readyForSubmit(d)
	d.docs.hasRequired = false
	d.docs.missing = []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE}
	srv := newTestServer(d)

	_, err := srv.SubmitApplication(t.Context(), &kycv1.SubmitApplicationRequest{UserId: TEST_USER_ID})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Equal(t, "required documents are missing: passport, selfie", st.Message())
}

func TestKycServer_GetApplicationByID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "found",
			setup: func(d *testDeps) { d.apps.getByIDResult = fullApplication() },
			want:  fullApplicationProto()},
		{name: "not found",
			setup:   func(d *testDeps) { d.apps.getByIDErr = domain.ErrApplicationNotFound },
			wantErr: domain.ErrApplicationMissing, wantCode: codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.GetApplicationByID(t.Context(), &kycv1.GetApplicationByIDRequest{ApplicationId: TEST_APPLICATION_ID})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
			assert.Equal(t, []uuid.UUID{uuid.MustParse(TEST_APPLICATION_ID)}, d.apps.getByIDCalls)
		})
	}
}

func TestKycServer_GetApplicationByUserID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "found",
			setup: func(d *testDeps) { d.apps.latestResult = fullApplication() },
			want:  fullApplicationProto()},
		{name: "not found",
			setup:   func(d *testDeps) { d.apps.latestErr = domain.ErrApplicationNotFound },
			wantErr: domain.ErrApplicationMissing, wantCode: codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.GetApplicationByUserID(t.Context(), &kycv1.GetApplicationByUserIDRequest{UserId: TEST_USER_ID})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
			assert.Equal(t, []uuid.UUID{uuid.MustParse(TEST_USER_ID)}, d.apps.latestCalls)
		})
	}
}

// R1, R5: фильтр, который сервер передаёт в репозиторий
func TestKycServer_ListApplications_Filter(t *testing.T) {
	officerID := uuid.MustParse(TEST_OFFICER_ID)
	tests := []struct {
		name string
		req  *kycv1.ListApplicationsRequest
		want models.ApplicationFilter
	}{
		{name: "unknown statuses dropped, defaults applied",
			req: &kycv1.ListApplicationsRequest{
				Statuses:   []kycv1.ApplicationStatus{kycv1.ApplicationStatus_APPLICATION_STATUS_IN_REVIEW, kycv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED, kycv1.ApplicationStatus(99)},
				Unassigned: true,
				Limit:      0,
			},
			want: models.ApplicationFilter{Statuses: []models.ApplicationStatus{models.STATUS_IN_REVIEW}, ReviewerID: nil, Unassigned: true, Limit: 20, Offset: 0}},
		{name: "all seven statuses in order",
			req: &kycv1.ListApplicationsRequest{
				Statuses: []kycv1.ApplicationStatus{
					kycv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED,
					kycv1.ApplicationStatus_APPLICATION_STATUS_AUTO_CHECKS,
					kycv1.ApplicationStatus_APPLICATION_STATUS_AUTO_REJECTED,
					kycv1.ApplicationStatus_APPLICATION_STATUS_IN_REVIEW,
					kycv1.ApplicationStatus_APPLICATION_STATUS_DOCS_REQUESTED,
					kycv1.ApplicationStatus_APPLICATION_STATUS_APPROVED,
					kycv1.ApplicationStatus_APPLICATION_STATUS_REJECTED,
				},
				Limit: 10,
			},
			want: models.ApplicationFilter{
				Statuses: []models.ApplicationStatus{
					models.STATUS_SUBMITTED, models.STATUS_AUTO_CHECKS, models.STATUS_AUTO_REJECTED, models.STATUS_IN_REVIEW,
					models.STATUS_DOCS_REQUESTED, models.STATUS_APPROVED, models.STATUS_REJECTED,
				},
				Limit: 10,
			}},
		{name: "reviewer_id parsed, limit and offset passed",
			req:  &kycv1.ListApplicationsRequest{Statuses: []kycv1.ApplicationStatus{kycv1.ApplicationStatus_APPLICATION_STATUS_APPROVED}, ReviewerId: TEST_OFFICER_ID, Limit: 50, Offset: 10},
			want: models.ApplicationFilter{Statuses: []models.ApplicationStatus{models.STATUS_APPROVED}, ReviewerID: &officerID, Limit: 50, Offset: 10}},
		{name: "limit above ceiling and negative offset normalized",
			req:  &kycv1.ListApplicationsRequest{Statuses: []kycv1.ApplicationStatus{kycv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED}, Limit: 500, Offset: -5},
			want: models.ApplicationFilter{Statuses: []models.ApplicationStatus{models.STATUS_SUBMITTED}, Limit: 100, Offset: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			srv := newTestServer(d)

			_, err := srv.ListApplications(t.Context(), tt.req)

			require.NoError(t, err)
			assert.Equal(t, []models.ApplicationFilter{tt.want}, d.apps.listCalls)
		})
	}
}

// R5: пустой или полностью отброшенный список статусов - пустой Statuses (nil и [] не различаются)
func TestKycServer_ListApplications_EmptyStatuses(t *testing.T) {
	tests := []struct {
		name     string
		statuses []kycv1.ApplicationStatus
	}{
		{name: "nil statuses", statuses: nil},
		{name: "only unknown statuses", statuses: []kycv1.ApplicationStatus{kycv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED, kycv1.ApplicationStatus(99)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			srv := newTestServer(d)

			_, err := srv.ListApplications(t.Context(), &kycv1.ListApplicationsRequest{Statuses: tt.statuses, Limit: 5, Offset: 1})

			require.NoError(t, err)
			require.Len(t, d.apps.listCalls, 1)
			filter := d.apps.listCalls[0]
			assert.Empty(t, filter.Statuses)
			assert.Nil(t, filter.ReviewerID)
			assert.False(t, filter.Unassigned)
			assert.Equal(t, 5, filter.Limit)
			assert.Equal(t, 1, filter.Offset)
		})
	}
}

// R5: ответ - заявки в порядке репозитория
func TestKycServer_ListApplications_Response(t *testing.T) {
	tests := []struct {
		name       string
		listResult []*models.Application
		want       []*kycv1.Application
	}{
		{name: "two applications in order",
			listResult: []*models.Application{minimalApplication(TEST_APPLICATION_ID_2), fullApplication()},
			want:       []*kycv1.Application{minimalApplicationProto(TEST_APPLICATION_ID_2), fullApplicationProto()}},
		{name: "empty", listResult: []*models.Application{}, want: []*kycv1.Application{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			d.apps.listResult = tt.listResult
			srv := newTestServer(d)

			got, err := srv.ListApplications(t.Context(), &kycv1.ListApplicationsRequest{})

			require.NoError(t, err)
			require.NotNil(t, got)
			assertProtoEqual(t, &kycv1.ListApplicationsResponse{Applications: tt.want}, got)
			assert.Len(t, got.GetApplications(), len(tt.want))
		})
	}
}

func TestKycServer_ClaimApplication(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "claimed",
			setup: func(d *testDeps) { d.apps.claimResult = fullApplication() },
			want:  fullApplicationProto()},
		{name: "already claimed",
			setup:   func(d *testDeps) { d.apps.claimErr = domain.ErrAlreadyClaimed },
			wantErr: domain.ErrClaimedByOther, wantCode: codes.FailedPrecondition},
		{name: "not found",
			setup:   func(d *testDeps) { d.apps.claimErr = domain.ErrApplicationNotFound },
			wantErr: domain.ErrApplicationMissing, wantCode: codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.ClaimApplication(t.Context(), &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

// R7: в Claim приходят разобранные application_id и officer_id
func TestKycServer_ClaimApplication_Args(t *testing.T) {
	d := newTestDeps()
	srv := newTestServer(d)

	_, err := srv.ClaimApplication(t.Context(), &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID})

	require.NoError(t, err)
	want := []claimCall{{ApplicationID: uuid.MustParse(TEST_APPLICATION_ID), OfficerID: uuid.MustParse(TEST_OFFICER_ID)}}
	assert.Equal(t, want, d.apps.claimCalls)
}

func TestKycServer_ApproveApplication(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "approved",
			setup: func(d *testDeps) { d.apps.transitionResult = fullApplication() },
			want:  fullApplicationProto()},
		{name: "status conflict",
			setup:   func(d *testDeps) { d.apps.transitionErr = domain.ErrStatusConflict },
			wantErr: domain.ErrTransitionForbidden, wantCode: codes.FailedPrecondition},
		{name: "application not found",
			setup:   func(d *testDeps) { d.apps.transitionErr = domain.ErrApplicationNotFound },
			wantErr: domain.ErrApplicationMissing, wantCode: codes.NotFound},
		{name: "customer not found on set status (wrapped)",
			setup: func(d *testDeps) {
				d.apps.transitionResult = minimalApplication(TEST_APPLICATION_ID)
				d.customer.setStatusErr = domain.ErrCustomerNotFound
			},
			wantErr: domain.ErrCustomerMissing, wantCode: codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.ApproveApplication(t.Context(), &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

func TestKycServer_RejectApplication(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "rejected", reason: TEST_REASON,
			setup: func(d *testDeps) { d.apps.transitionResult = fullApplication() },
			want:  fullApplicationProto()},
		{name: "reason of spaces", reason: "   ",
			setup:   func(d *testDeps) {},
			wantErr: domain.ErrReasonIsRequired, wantCode: codes.InvalidArgument},
		{name: "empty reason", reason: "",
			setup:   func(d *testDeps) {},
			wantErr: domain.ErrReasonIsRequired, wantCode: codes.InvalidArgument},
		{name: "status conflict", reason: TEST_REASON,
			setup:   func(d *testDeps) { d.apps.transitionErr = domain.ErrStatusConflict },
			wantErr: domain.ErrTransitionForbidden, wantCode: codes.FailedPrecondition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.RejectApplication(t.Context(), &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Reason: tt.reason})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

func TestKycServer_RequestMoreDocuments(t *testing.T) {
	tests := []struct {
		name     string
		types    []kycv1.DocumentType
		setup    func(d *testDeps)
		want     *kycv1.Application
		wantErr  error
		wantCode codes.Code
	}{
		{name: "requested", types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS},
			setup: func(d *testDeps) { d.apps.transitionResult = fullApplication() },
			want:  fullApplicationProto()},
		{name: "unknown type", types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT, kycv1.DocumentType(99)},
			setup:   func(d *testDeps) {},
			wantErr: domain.ErrTypesRequired, wantCode: codes.InvalidArgument},
		{name: "unspecified type", types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED},
			setup:   func(d *testDeps) {},
			wantErr: domain.ErrTypesRequired, wantCode: codes.InvalidArgument},
		{name: "empty types", types: nil,
			setup:   func(d *testDeps) {},
			wantErr: domain.ErrTypesRequired, wantCode: codes.InvalidArgument},
		{name: "status conflict", types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_SELFIE},
			setup:   func(d *testDeps) { d.apps.transitionErr = domain.ErrStatusConflict },
			wantErr: domain.ErrTransitionForbidden, wantCode: codes.FailedPrecondition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			tt.setup(d)
			srv := newTestServer(d)

			got, err := srv.RequestMoreDocuments(t.Context(), &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Types: tt.types, Comment: TEST_COMMENT})

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

// R6, R7: что Approve/Reject/RequestMoreDocuments передают в Transition
func TestKycServer_Review_TransitionArgs(t *testing.T) {
	applicationID := uuid.MustParse(TEST_APPLICATION_ID)
	officerID := uuid.MustParse(TEST_OFFICER_ID)
	inReview := []models.ApplicationStatus{models.STATUS_IN_REVIEW}
	tests := []struct {
		name string
		call func(ctx context.Context, srv *KycServer) error
		want []transitionCall
	}{
		{name: "approve",
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.ApproveApplication(ctx, &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT})
				return err
			},
			want: []transitionCall{{ApplicationID: applicationID, AllowedFrom: inReview,
				Transition: models.Transition{To: models.STATUS_APPROVED, ActorID: &officerID, Comment: TEST_COMMENT, Decided: true}}}},
		{name: "reject",
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.RejectApplication(ctx, &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Reason: TEST_REASON})
				return err
			},
			want: []transitionCall{{ApplicationID: applicationID, AllowedFrom: inReview,
				Transition: models.Transition{To: models.STATUS_REJECTED, ActorID: &officerID, Comment: TEST_REASON, Reason: TEST_REASON, Decided: true}}}},
		{name: "request more documents keeps type order",
			call: func(ctx context.Context, srv *KycServer) error {
				_, err := srv.RequestMoreDocuments(ctx, &kycv1.RequestMoreDocumentsRequest{
					ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: TEST_COMMENT,
					Types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS, kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT, kycv1.DocumentType_DOCUMENT_TYPE_SELFIE},
				})
				return err
			},
			want: []transitionCall{{ApplicationID: applicationID, AllowedFrom: inReview,
				Transition: models.Transition{
					To: models.STATUS_DOCS_REQUESTED, ActorID: &officerID, Comment: TEST_COMMENT, Reason: TEST_COMMENT,
					RequestedTypes: []models.DocumentType{models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE},
					Decided:        false,
				}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			d.apps.transitionResult = minimalApplication(TEST_APPLICATION_ID)
			srv := newTestServer(d)

			err := tt.call(t.Context(), srv)

			require.NoError(t, err)
			assert.Equal(t, tt.want, d.apps.transitionCalls)
		})
	}
}

// R2, R6: отвергнутые сервером запросы до Transition не доходят
func TestKycServer_Review_RejectedBeforeTransition(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, srv *KycServer) error
	}{
		{name: "reject with blank reason", call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.RejectApplication(ctx, &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Reason: "   "})
			return err
		}},
		{name: "request documents with unknown type", call: func(ctx context.Context, srv *KycServer) error {
			_, err := srv.RequestMoreDocuments(ctx, &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Types: []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT, kycv1.DocumentType(99)}})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			srv := newTestServer(d)

			err := tt.call(t.Context(), srv)

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Empty(t, d.apps.transitionCalls)
		})
	}
}

func TestKycServer_AddSanctionEntry(t *testing.T) {
	sanctionID := uuid.MustParse(TEST_SANCTION_ID)
	tests := []struct {
		name      string
		req       *kycv1.AddSanctionEntryRequest
		addResult *models.SanctionEntry
		addErr    error
		wantAdd   []models.SanctionEntry
		want      *kycv1.SanctionEntry
		wantErr   error
		wantCode  codes.Code
	}{
		{name: "trimmed entry stored, response from repository",
			req:       &kycv1.AddSanctionEntryRequest{FirstName: " Ivan ", LastName: "Petrov", BirthDate: &date.Date{Year: 1980, Month: 1, Day: 15}, Source: "OFAC"},
			addResult: &models.SanctionEntry{ID: sanctionID, FirstName: "Ivan", LastName: "Petrov", BirthDate: time.Date(1980, 1, 15, 0, 0, 0, 0, time.UTC), Source: "OFAC"},
			wantAdd:   []models.SanctionEntry{{FirstName: "Ivan", LastName: "Petrov", BirthDate: time.Date(1980, 1, 15, 0, 0, 0, 0, time.UTC), Source: "OFAC"}},
			want:      &kycv1.SanctionEntry{Id: TEST_SANCTION_ID, FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1980, Month: 1, Day: 15}, Source: "OFAC"}},
		{name: "response mirrors repository record, not request",
			req:       &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1980, Month: 1, Day: 15}, Source: " UN "},
			addResult: &models.SanctionEntry{ID: sanctionID, FirstName: "Oleg", LastName: "Sidorov", BirthDate: time.Date(1990, 12, 31, 0, 0, 0, 0, time.UTC), Source: "EU"},
			wantAdd:   []models.SanctionEntry{{FirstName: "Ivan", LastName: "Petrov", BirthDate: time.Date(1980, 1, 15, 0, 0, 0, 0, time.UTC), Source: "UN"}},
			want:      &kycv1.SanctionEntry{Id: TEST_SANCTION_ID, FirstName: "Oleg", LastName: "Sidorov", BirthDate: &date.Date{Year: 1990, Month: 12, Day: 31}, Source: "EU"}},
		{name: "duplicate entry",
			req:     &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1980, Month: 1, Day: 15}, Source: "OFAC"},
			addErr:  domain.ErrSanctionEntryExists,
			wantErr: domain.ErrSanctionExists, wantCode: codes.AlreadyExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			d.sanctions.addResult = tt.addResult
			d.sanctions.addErr = tt.addErr
			srv := newTestServer(d)

			got, err := srv.AddSanctionEntry(t.Context(), tt.req)

			if tt.wantErr != nil {
				requireStatus(t, err, tt.wantErr, tt.wantCode)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAdd, d.sanctions.addCalls)
			assertProtoEqual(t, tt.want, got)
		})
	}
}

func TestKycServer_ListSanctions(t *testing.T) {
	entries := []models.SanctionEntry{
		{ID: uuid.MustParse(TEST_SANCTION_ID), FirstName: "Ivan", LastName: "Petrov", BirthDate: time.Date(1980, 1, 15, 0, 0, 0, 0, time.UTC), Source: "OFAC"},
		{ID: uuid.MustParse(TEST_APPLICATION_ID_2), FirstName: "Oleg", LastName: "Sidorov", BirthDate: time.Date(1990, 12, 31, 0, 0, 0, 0, time.UTC), Source: "EU"},
	}
	wantEntries := []*kycv1.SanctionEntry{
		{Id: TEST_SANCTION_ID, FirstName: "Ivan", LastName: "Petrov", BirthDate: &date.Date{Year: 1980, Month: 1, Day: 15}, Source: "OFAC"},
		{Id: TEST_APPLICATION_ID_2, FirstName: "Oleg", LastName: "Sidorov", BirthDate: &date.Date{Year: 1990, Month: 12, Day: 31}, Source: "EU"},
	}
	tests := []struct {
		name       string
		req        *kycv1.ListSanctionsRequest
		listResult []models.SanctionEntry
		wantPage   []pageCall
		want       []*kycv1.SanctionEntry
	}{
		{name: "default limit", req: &kycv1.ListSanctionsRequest{Limit: 0, Offset: 0},
			listResult: entries, wantPage: []pageCall{{Limit: 20, Offset: 0}}, want: wantEntries},
		{name: "explicit page", req: &kycv1.ListSanctionsRequest{Limit: 5, Offset: 3},
			listResult: entries, wantPage: []pageCall{{Limit: 5, Offset: 3}}, want: wantEntries},
		{name: "empty", req: &kycv1.ListSanctionsRequest{Limit: 5, Offset: 3},
			listResult: []models.SanctionEntry{}, wantPage: []pageCall{{Limit: 5, Offset: 3}}, want: []*kycv1.SanctionEntry{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDeps()
			d.sanctions.listResult = tt.listResult
			srv := newTestServer(d)

			got, err := srv.ListSanctions(t.Context(), tt.req)

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantPage, d.sanctions.listCalls)
			assertProtoEqual(t, &kycv1.ListSanctionsResponse{Entries: tt.want}, got)
			assert.Len(t, got.GetEntries(), len(tt.want))
		})
	}
}
