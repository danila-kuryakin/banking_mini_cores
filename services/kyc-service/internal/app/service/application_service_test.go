package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// newTestApplicationService собирает сервис на фейках
func newTestApplicationService(apps *fakeApplicationRepository, customer *fakeCustomerProfiles, docs *fakeDocuments) *ApplicationService {
	return NewApplicationService(
		&repository.Repository{Application: apps, Sanctions: &fakeSanctionsRepository{}},
		&client.Clients{Customer: customer, Documents: docs},
	)
}

// filledProfile возвращает заполненную анкету в статусе profile_filled
func filledProfile() *models.CustomerProfile {
	return &models.CustomerProfile{
		FirstName: TEST_FIRST_NAME,
		LastName:  TEST_LAST_NAME,
		BirthDate: testBirthDate(),
		Status:    models.CUSTOMER_STATUS_PROFILE_FILLED,
	}
}

// profileWith возвращает анкету после правки
func profileWith(edit func(p *models.CustomerProfile)) *models.CustomerProfile {
	profile := filledProfile()
	edit(profile)
	return profile
}

// previousApplication возвращает прошлую заявку клиента в заданном статусе
func previousApplication(status models.ApplicationStatus) *models.Application {
	return &models.Application{ID: TEST_PREVIOUS_APPLICATION_ID, UserID: TEST_USER_ID, Status: status}
}

// docsRequestedApplication возвращает заявку, по которой офицер запросил документы
func docsRequestedApplication(requested ...models.DocumentType) *models.Application {
	return &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_DOCS_REQUESTED, RequestedTypes: requested}
}

// submitFixture - настройка зависимостей одного кейса Submit
type submitFixture struct {
	latest         *models.Application
	latestStored   bool
	latestErr      error
	profile        *models.CustomerProfile
	getProfileErr  error
	hasRequired    bool
	missing        []models.DocumentType
	hasRequiredErr error
	confirmed      []models.DocumentType
	confirmedErr   error
	createErr      error
	transitionErr  error
	setStatusErr   error
}

// build создаёт фейки по настройке кейса
func (f submitFixture) build() (*fakeApplicationRepository, *fakeCustomerProfiles, *fakeDocuments) {
	apps := newFakeApplicationRepository()
	apps.latest = f.latest
	apps.latestErr = f.latestErr
	apps.createErr = f.createErr
	apps.transitionErr = f.transitionErr
	if f.latestStored {
		apps.apps[f.latest.ID] = *f.latest
	}
	customer := &fakeCustomerProfiles{profile: f.profile, getProfileErr: f.getProfileErr, setStatusErr: f.setStatusErr}
	docs := &fakeDocuments{
		hasRequired:    f.hasRequired,
		missing:        f.missing,
		hasRequiredErr: f.hasRequiredErr,
		confirmed:      f.confirmed,
		confirmedErr:   f.confirmedErr,
	}
	return apps, customer, docs
}

func TestApplicationService_Submit(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)
	userID := TEST_USER_ID

	createdApp := &models.Application{
		ID:        TEST_APPLICATION_ID,
		UserID:    userID,
		Status:    models.STATUS_SUBMITTED,
		FirstName: TEST_FIRST_NAME,
		LastName:  TEST_LAST_NAME,
		BirthDate: testBirthDate(),
	}
	resubmittedApp := &models.Application{ID: TEST_APPLICATION_ID, UserID: userID, Status: models.STATUS_SUBMITTED}
	wantCreate := []models.Application{{UserID: userID, FirstName: TEST_FIRST_NAME, LastName: TEST_LAST_NAME, BirthDate: testBirthDate()}}
	wantOnKyc := []setStatusCall{{UserID: userID, To: models.CUSTOMER_STATUS_ON_KYC, Reason: domain.REASON_APPLICATION_SUBMITTED, ActorID: nil}}
	wantResubmit := []transitionCall{{
		ApplicationID: TEST_APPLICATION_ID,
		Transition:    models.Transition{To: models.STATUS_SUBMITTED},
		AllowedFrom:   []models.ApplicationStatus{models.STATUS_DOCS_REQUESTED},
	}}
	wantUser := []uuid.UUID{userID}

	tests := []struct {
		name           string
		fixture        submitFixture
		want           *models.Application
		wantErr        error
		wantGetProfile []uuid.UUID
		wantHasReq     []uuid.UUID
		wantConfirmed  []uuid.UUID
		wantCreate     []models.Application
		wantTransition []transitionCall
		wantSetStatus  []setStatusCall
	}{
		{
			name:           "no previous application creates new one",
			fixture:        submitFixture{profile: filledProfile(), hasRequired: true},
			want:           createdApp,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
			wantSetStatus:  wantOnKyc,
		},
		{
			name:           "previous auto_rejected creates new one",
			fixture:        submitFixture{latest: previousApplication(models.STATUS_AUTO_REJECTED), profile: filledProfile(), hasRequired: true},
			want:           createdApp,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
			wantSetStatus:  wantOnKyc,
		},
		{
			name:           "previous approved creates new one",
			fixture:        submitFixture{latest: previousApplication(models.STATUS_APPROVED), profile: filledProfile(), hasRequired: true},
			want:           createdApp,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
			wantSetStatus:  wantOnKyc,
		},
		{
			name:           "previous rejected creates new one",
			fixture:        submitFixture{latest: previousApplication(models.STATUS_REJECTED), profile: filledProfile(), hasRequired: true},
			want:           createdApp,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
			wantSetStatus:  wantOnKyc,
		},
		{
			name:    "previous submitted is active",
			fixture: submitFixture{latest: previousApplication(models.STATUS_SUBMITTED), profile: filledProfile(), hasRequired: true},
			wantErr: domain.ErrActiveApplicationExists,
		},
		{
			name:    "previous auto_checks is active",
			fixture: submitFixture{latest: previousApplication(models.STATUS_AUTO_CHECKS), profile: filledProfile(), hasRequired: true},
			wantErr: domain.ErrActiveApplicationExists,
		},
		{
			name:    "previous in_review is active",
			fixture: submitFixture{latest: previousApplication(models.STATUS_IN_REVIEW), profile: filledProfile(), hasRequired: true},
			wantErr: domain.ErrActiveApplicationExists,
		},
		{
			name: "docs_requested resubmits with requested types confirmed",
			fixture: submitFixture{
				latest:       docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS),
				latestStored: true,
				hasRequired:  true,
				confirmed:    []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PROOF_OF_ADDRESS},
			},
			want:           resubmittedApp,
			wantHasReq:     wantUser,
			wantConfirmed:  wantUser,
			wantTransition: wantResubmit,
		},
		{
			name:           "docs_requested without requested types skips confirmed types",
			fixture:        submitFixture{latest: docsRequestedApplication(), latestStored: true, hasRequired: true},
			want:           resubmittedApp,
			wantHasReq:     wantUser,
			wantTransition: wantResubmit,
		},
		{
			name:       "docs_requested required documents check fails",
			fixture:    submitFixture{latest: docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS), latestStored: true, hasRequiredErr: errBoom},
			wantErr:    errBoom,
			wantHasReq: wantUser,
		},
		{
			name: "docs_requested confirmed types fails",
			fixture: submitFixture{
				latest:       docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS),
				latestStored: true,
				hasRequired:  true,
				confirmedErr: errBoom,
			},
			wantErr:       errBoom,
			wantHasReq:    wantUser,
			wantConfirmed: wantUser,
		},
		{
			name:           "docs_requested transition conflicts",
			fixture:        submitFixture{latest: docsRequestedApplication(), latestStored: true, hasRequired: true, transitionErr: domain.ErrStatusConflict},
			wantErr:        domain.ErrStatusConflict,
			wantHasReq:     wantUser,
			wantTransition: wantResubmit,
		},
		{
			// заявки нет в хранилище - фейк отвечает not found
			name:           "docs_requested transition not found",
			fixture:        submitFixture{latest: docsRequestedApplication(), hasRequired: true},
			wantErr:        domain.ErrApplicationNotFound,
			wantHasReq:     wantUser,
			wantTransition: wantResubmit,
		},
		{
			name:    "latest application lookup fails",
			fixture: submitFixture{latestErr: errBoom, profile: filledProfile(), hasRequired: true},
			wantErr: errBoom,
		},
		{
			name:           "customer status new",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.Status = models.CUSTOMER_STATUS_NEW }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "customer status on_kyc",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.Status = models.CUSTOMER_STATUS_ON_KYC }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "customer status active",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.Status = models.CUSTOMER_STATUS_ACTIVE }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "customer status rejected",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.Status = models.CUSTOMER_STATUS_REJECTED }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "customer status blocked",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.Status = models.CUSTOMER_STATUS_BLOCKED }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "profile without first name",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.FirstName = "" }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "profile without last name",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.LastName = "" }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			name:           "profile without birth date",
			fixture:        submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.BirthDate = nil }), hasRequired: true},
			wantErr:        domain.ErrProfileNotFilled,
			wantGetProfile: wantUser,
		},
		{
			// фейк без профиля отвечает ErrCustomerNotFound
			name:           "customer not found",
			fixture:        submitFixture{hasRequired: true},
			wantErr:        domain.ErrCustomerNotFound,
			wantGetProfile: wantUser,
		},
		{
			name:           "get profile fails",
			fixture:        submitFixture{getProfileErr: errBoom, hasRequired: true},
			wantErr:        errBoom,
			wantGetProfile: wantUser,
		},
		{
			name:           "new application required documents check fails",
			fixture:        submitFixture{profile: filledProfile(), hasRequiredErr: errBoom},
			wantErr:        errBoom,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
		},
		{
			name:           "new application documents missing",
			fixture:        submitFixture{profile: filledProfile(), missing: []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT}},
			wantErr:        domain.ErrDocumentsMissing,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
		},
		{
			name:           "create races with active application",
			fixture:        submitFixture{profile: filledProfile(), hasRequired: true, createErr: domain.ErrActiveApplicationExists},
			wantErr:        domain.ErrActiveApplicationExists,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
		},
		{
			name:           "create fails",
			fixture:        submitFixture{profile: filledProfile(), hasRequired: true, createErr: errBoom},
			wantErr:        errBoom,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
		},
		{
			name:           "set status customer not found",
			fixture:        submitFixture{profile: filledProfile(), hasRequired: true, setStatusErr: domain.ErrCustomerNotFound},
			wantErr:        domain.ErrCustomerNotFound,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
			wantSetStatus:  wantOnKyc,
		},
		{
			name:           "set status fails",
			fixture:        submitFixture{profile: filledProfile(), hasRequired: true, setStatusErr: errBoom},
			wantErr:        errBoom,
			wantGetProfile: wantUser,
			wantHasReq:     wantUser,
			wantCreate:     wantCreate,
			wantSetStatus:  wantOnKyc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps, customer, docs := tt.fixture.build()
			svc := newTestApplicationService(apps, customer, docs)

			got, err := svc.Submit(t.Context(), userID)

			assert.Equal(t, wantUser, apps.latestCalls)
			assert.Equal(t, tt.wantGetProfile, customer.getProfileCalls)
			assert.Equal(t, tt.wantHasReq, docs.hasRequiredCalls)
			assert.Equal(t, tt.wantConfirmed, docs.confirmedCalls)
			assert.Equal(t, tt.wantCreate, apps.createCalls)
			assert.Equal(t, tt.wantTransition, apps.transitionCalls)
			assert.Equal(t, tt.wantSetStatus, customer.setStatusCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_Submit_DocumentsMissingMessage(t *testing.T) {
	tests := []struct {
		name    string
		fixture submitFixture
		wantMsg string
	}{
		{
			name:    "new application one required missing",
			fixture: submitFixture{profile: filledProfile(), missing: []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT}},
			wantMsg: "required documents are missing: passport",
		},
		{
			name:    "new application two required missing",
			fixture: submitFixture{profile: filledProfile(), missing: []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE}},
			wantMsg: "required documents are missing: passport, selfie",
		},
		{
			name: "resubmit required missing",
			fixture: submitFixture{
				latest:       docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS),
				latestStored: true,
				missing:      []models.DocumentType{models.DOCUMENT_TYPE_SELFIE},
			},
			wantMsg: "required documents are missing: selfie",
		},
		{
			name: "resubmit requested type not confirmed",
			fixture: submitFixture{
				latest:       docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS),
				latestStored: true,
				hasRequired:  true,
				confirmed:    []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE},
			},
			wantMsg: "required documents are missing: proof_of_address",
		},
		{
			// подтверждён только паспорт - остальные запрошенные в порядке запроса
			name: "resubmit requested types partially confirmed",
			fixture: submitFixture{
				latest:       docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PASSPORT),
				latestStored: true,
				hasRequired:  true,
				confirmed:    []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT},
			},
			wantMsg: "required documents are missing: proof_of_address, selfie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps, customer, docs := tt.fixture.build()
			svc := newTestApplicationService(apps, customer, docs)

			got, err := svc.Submit(t.Context(), TEST_USER_ID)

			require.ErrorIs(t, err, domain.ErrDocumentsMissing)
			assert.EqualError(t, err, tt.wantMsg)
			assert.Nil(t, got)
		})
	}
}

func TestApplicationService_Submit_NoWritesOnEarlyError(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)

	tests := []struct {
		name    string
		fixture submitFixture
		wantErr error
	}{
		{
			name:    "latest lookup fails",
			fixture: submitFixture{latestErr: errBoom, profile: filledProfile(), hasRequired: true},
			wantErr: errBoom,
		},
		{
			name:    "active application exists",
			fixture: submitFixture{latest: previousApplication(models.STATUS_IN_REVIEW), latestStored: true, profile: filledProfile(), hasRequired: true},
			wantErr: domain.ErrActiveApplicationExists,
		},
		{
			name:    "customer not found",
			fixture: submitFixture{getProfileErr: domain.ErrCustomerNotFound, hasRequired: true},
			wantErr: domain.ErrCustomerNotFound,
		},
		{
			name:    "customer already on kyc",
			fixture: submitFixture{profile: profileWith(func(p *models.CustomerProfile) { p.Status = models.CUSTOMER_STATUS_ON_KYC }), hasRequired: true},
			wantErr: domain.ErrProfileNotFilled,
		},
		{
			name:    "required documents missing",
			fixture: submitFixture{profile: filledProfile(), missing: []models.DocumentType{models.DOCUMENT_TYPE_SELFIE}},
			wantErr: domain.ErrDocumentsMissing,
		},
		{
			name: "confirmed types fail on resubmit",
			fixture: submitFixture{
				latest:       docsRequestedApplication(models.DOCUMENT_TYPE_PROOF_OF_ADDRESS),
				latestStored: true,
				hasRequired:  true,
				confirmedErr: errBoom,
			},
			wantErr: errBoom,
		},
		{
			name:    "required documents check fails",
			fixture: submitFixture{profile: filledProfile(), hasRequiredErr: errBoom},
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps, customer, docs := tt.fixture.build()
			svc := newTestApplicationService(apps, customer, docs)

			_, err := svc.Submit(t.Context(), TEST_USER_ID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, []models.Application(nil), apps.createCalls)
			assert.Equal(t, []transitionCall(nil), apps.transitionCalls)
			assert.Equal(t, []setStatusCall(nil), customer.setStatusCalls)
		})
	}
}

func TestApplicationService_GetByID(t *testing.T) {
	stored := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_IN_REVIEW}

	tests := []struct {
		name          string
		applicationID uuid.UUID
		want          *models.Application
		wantErr       error
	}{
		{name: "found", applicationID: TEST_APPLICATION_ID, want: &stored},
		{name: "not found", applicationID: TEST_PREVIOUS_APPLICATION_ID, wantErr: domain.ErrApplicationNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository(stored)
			svc := newTestApplicationService(apps, &fakeCustomerProfiles{}, &fakeDocuments{})

			got, err := svc.GetByID(t.Context(), tt.applicationID)

			assert.Equal(t, []uuid.UUID{tt.applicationID}, apps.getByIDCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_GetLatestByUserID(t *testing.T) {
	latest := &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_APPROVED}

	tests := []struct {
		name    string
		latest  *models.Application
		want    *models.Application
		wantErr error
	}{
		{name: "found", latest: latest, want: latest},
		{name: "not found", wantErr: domain.ErrApplicationNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository()
			apps.latest = tt.latest
			svc := newTestApplicationService(apps, &fakeCustomerProfiles{}, &fakeDocuments{})

			got, err := svc.GetLatestByUserID(t.Context(), TEST_USER_ID)

			assert.Equal(t, []uuid.UUID{TEST_USER_ID}, apps.latestCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_Claim(t *testing.T) {
	officerID := TEST_OFFICER_ID
	claimed := &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_IN_REVIEW, Reviewer: &officerID}

	tests := []struct {
		name        string
		claimResult *models.Application
		claimErr    error
		want        *models.Application
		wantErr     error
	}{
		{name: "claimed", claimResult: claimed, want: claimed},
		{name: "already claimed", claimErr: domain.ErrAlreadyClaimed, wantErr: domain.ErrAlreadyClaimed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository()
			apps.claimResult = tt.claimResult
			apps.claimErr = tt.claimErr
			svc := newTestApplicationService(apps, &fakeCustomerProfiles{}, &fakeDocuments{})

			got, err := svc.Claim(t.Context(), TEST_APPLICATION_ID, officerID)

			assert.Equal(t, []claimCall{{ApplicationID: TEST_APPLICATION_ID, OfficerID: officerID}}, apps.claimCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_List(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)
	reviewerID := TEST_OFFICER_ID
	twoApps := []*models.Application{
		{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_IN_REVIEW},
		{ID: TEST_PREVIOUS_APPLICATION_ID, UserID: TEST_OTHER_USER_ID, Status: models.STATUS_SUBMITTED},
	}

	tests := []struct {
		name       string
		filter     models.ApplicationFilter
		listResult []*models.Application
		listErr    error
		wantFilter models.ApplicationFilter
		want       []*models.Application
		wantErr    error
	}{
		{name: "zero limit and negative offset", filter: models.ApplicationFilter{Limit: 0, Offset: -5}, wantFilter: models.ApplicationFilter{Limit: 20, Offset: 0}},
		{name: "negative limit", filter: models.ApplicationFilter{Limit: -1}, wantFilter: models.ApplicationFilter{Limit: 20}},
		{name: "minimal limit", filter: models.ApplicationFilter{Limit: 1}, wantFilter: models.ApplicationFilter{Limit: 1}},
		{name: "middle limit", filter: models.ApplicationFilter{Limit: 50}, wantFilter: models.ApplicationFilter{Limit: 50}},
		{name: "max limit", filter: models.ApplicationFilter{Limit: 100}, wantFilter: models.ApplicationFilter{Limit: 100}},
		{name: "over max limit", filter: models.ApplicationFilter{Limit: 101}, wantFilter: models.ApplicationFilter{Limit: 100}},
		{name: "zero offset", filter: models.ApplicationFilter{Limit: 10, Offset: 0}, wantFilter: models.ApplicationFilter{Limit: 10, Offset: 0}},
		{name: "positive offset", filter: models.ApplicationFilter{Limit: 10, Offset: 7}, wantFilter: models.ApplicationFilter{Limit: 10, Offset: 7}},
		{name: "offset minus one", filter: models.ApplicationFilter{Limit: 10, Offset: -1}, wantFilter: models.ApplicationFilter{Limit: 10, Offset: 0}},
		{
			name: "filter fields pass through",
			filter: models.ApplicationFilter{
				Statuses:   []models.ApplicationStatus{models.STATUS_IN_REVIEW, models.STATUS_SUBMITTED},
				ReviewerID: &reviewerID,
				Unassigned: true,
				Limit:      10,
				Offset:     3,
			},
			listResult: twoApps,
			wantFilter: models.ApplicationFilter{
				Statuses:   []models.ApplicationStatus{models.STATUS_IN_REVIEW, models.STATUS_SUBMITTED},
				ReviewerID: &reviewerID,
				Unassigned: true,
				Limit:      10,
				Offset:     3,
			},
			want: twoApps,
		},
		{name: "empty result", filter: models.ApplicationFilter{Limit: 10}, listResult: []*models.Application{}, wantFilter: models.ApplicationFilter{Limit: 10}, want: []*models.Application{}},
		{name: "nil result", filter: models.ApplicationFilter{Limit: 10}, wantFilter: models.ApplicationFilter{Limit: 10}, want: nil},
		{name: "repository fails", filter: models.ApplicationFilter{Limit: 10}, listErr: errBoom, wantFilter: models.ApplicationFilter{Limit: 10}, wantErr: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository()
			apps.listResult = tt.listResult
			apps.listErr = tt.listErr
			svc := newTestApplicationService(apps, &fakeCustomerProfiles{}, &fakeDocuments{})

			got, err := svc.List(t.Context(), tt.filter)

			assert.Equal(t, []models.ApplicationFilter{tt.wantFilter}, apps.listCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_Approve(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)
	officerID := TEST_OFFICER_ID
	inReview := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_IN_REVIEW}
	otherOwner := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_OTHER_USER_ID, Status: models.STATUS_IN_REVIEW}
	submitted := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_SUBMITTED}

	approvedCall := func(comment string) []transitionCall {
		return []transitionCall{{
			ApplicationID: TEST_APPLICATION_ID,
			Transition:    models.Transition{To: models.STATUS_APPROVED, ActorID: &officerID, Comment: comment, Decided: true},
			AllowedFrom:   []models.ApplicationStatus{models.STATUS_IN_REVIEW},
		}}
	}
	activeCall := func(userID uuid.UUID) []setStatusCall {
		return []setStatusCall{{UserID: userID, To: models.CUSTOMER_STATUS_ACTIVE, Reason: domain.REASON_APPLICATION_APPROVED, ActorID: &officerID}}
	}

	tests := []struct {
		name           string
		stored         []models.Application
		comment        string
		transitionErr  error
		setStatusErr   error
		want           *models.Application
		wantErr        error
		wantTransition []transitionCall
		wantSetStatus  []setStatusCall
	}{
		{
			name:           "approved",
			stored:         []models.Application{inReview},
			comment:        "ok",
			want:           &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_APPROVED},
			wantTransition: approvedCall("ok"),
			wantSetStatus:  activeCall(TEST_USER_ID),
		},
		{
			name:           "empty comment",
			stored:         []models.Application{inReview},
			comment:        "",
			want:           &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_APPROVED},
			wantTransition: approvedCall(""),
			wantSetStatus:  activeCall(TEST_USER_ID),
		},
		{
			// статус клиента двигается по владельцу заявки из ответа Transition
			name:           "customer taken from transitioned application",
			stored:         []models.Application{otherOwner},
			comment:        "ok",
			want:           &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_OTHER_USER_ID, Status: models.STATUS_APPROVED},
			wantTransition: approvedCall("ok"),
			wantSetStatus:  activeCall(TEST_OTHER_USER_ID),
		},
		{
			name:           "application not in review",
			stored:         []models.Application{submitted},
			comment:        "ok",
			wantErr:        domain.ErrStatusConflict,
			wantTransition: approvedCall("ok"),
		},
		{
			name:           "application not found",
			comment:        "ok",
			wantErr:        domain.ErrApplicationNotFound,
			wantTransition: approvedCall("ok"),
		},
		{
			name:           "transition fails",
			stored:         []models.Application{inReview},
			comment:        "ok",
			transitionErr:  errBoom,
			wantErr:        errBoom,
			wantTransition: approvedCall("ok"),
		},
		{
			name:           "set status customer not found",
			stored:         []models.Application{inReview},
			comment:        "ok",
			setStatusErr:   domain.ErrCustomerNotFound,
			wantErr:        domain.ErrCustomerNotFound,
			wantTransition: approvedCall("ok"),
			wantSetStatus:  activeCall(TEST_USER_ID),
		},
		{
			name:           "set status fails",
			stored:         []models.Application{inReview},
			comment:        "ok",
			setStatusErr:   errBoom,
			wantErr:        errBoom,
			wantTransition: approvedCall("ok"),
			wantSetStatus:  activeCall(TEST_USER_ID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository(tt.stored...)
			apps.transitionErr = tt.transitionErr
			customer := &fakeCustomerProfiles{setStatusErr: tt.setStatusErr}
			svc := newTestApplicationService(apps, customer, &fakeDocuments{})

			got, err := svc.Approve(t.Context(), TEST_APPLICATION_ID, officerID, tt.comment)

			assert.Equal(t, tt.wantTransition, apps.transitionCalls)
			assert.Equal(t, tt.wantSetStatus, customer.setStatusCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_Reject(t *testing.T) {
	const reason = "fake passport"
	officerID := TEST_OFFICER_ID
	inReview := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_IN_REVIEW}
	submitted := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_SUBMITTED}

	rejectedCall := []transitionCall{{
		ApplicationID: TEST_APPLICATION_ID,
		Transition:    models.Transition{To: models.STATUS_REJECTED, ActorID: &officerID, Comment: reason, Reason: reason, Decided: true},
		AllowedFrom:   []models.ApplicationStatus{models.STATUS_IN_REVIEW},
	}}
	customerRejectedCall := []setStatusCall{{UserID: TEST_USER_ID, To: models.CUSTOMER_STATUS_REJECTED, Reason: reason, ActorID: &officerID}}

	tests := []struct {
		name           string
		stored         []models.Application
		reason         string
		setStatusErr   error
		want           *models.Application
		wantErr        error
		wantTransition []transitionCall
		wantSetStatus  []setStatusCall
	}{
		{
			name:           "rejected",
			stored:         []models.Application{inReview},
			reason:         reason,
			want:           &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_REJECTED, Reason: reason},
			wantTransition: rejectedCall,
			wantSetStatus:  customerRejectedCall,
		},
		{name: "empty reason", stored: []models.Application{inReview}, reason: "", wantErr: domain.ErrReasonRequired},
		{name: "spaces reason", stored: []models.Application{inReview}, reason: "   ", wantErr: domain.ErrReasonRequired},
		{name: "tab and newline reason", stored: []models.Application{inReview}, reason: "\t\n", wantErr: domain.ErrReasonRequired},
		{
			name:           "application not in review",
			stored:         []models.Application{submitted},
			reason:         reason,
			wantErr:        domain.ErrStatusConflict,
			wantTransition: rejectedCall,
		},
		{
			name:           "set status customer not found",
			stored:         []models.Application{inReview},
			reason:         reason,
			setStatusErr:   domain.ErrCustomerNotFound,
			wantErr:        domain.ErrCustomerNotFound,
			wantTransition: rejectedCall,
			wantSetStatus:  customerRejectedCall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository(tt.stored...)
			customer := &fakeCustomerProfiles{setStatusErr: tt.setStatusErr}
			svc := newTestApplicationService(apps, customer, &fakeDocuments{})

			got, err := svc.Reject(t.Context(), TEST_APPLICATION_ID, officerID, tt.reason)

			assert.Equal(t, []uuid.UUID(nil), apps.getByIDCalls)
			assert.Equal(t, tt.wantTransition, apps.transitionCalls)
			assert.Equal(t, tt.wantSetStatus, customer.setStatusCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplicationService_RequestDocuments(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)
	const comment = "need address"
	officerID := TEST_OFFICER_ID
	inReview := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_IN_REVIEW}
	submitted := models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_SUBMITTED}
	addressOnly := []models.DocumentType{models.DOCUMENT_TYPE_PROOF_OF_ADDRESS}
	allTypes := []models.DocumentType{models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, models.DOCUMENT_TYPE_PASSPORT}

	requestCall := func(types []models.DocumentType, comment string) []transitionCall {
		return []transitionCall{{
			ApplicationID: TEST_APPLICATION_ID,
			Transition: models.Transition{
				To:             models.STATUS_DOCS_REQUESTED,
				ActorID:        &officerID,
				Comment:        comment,
				Reason:         comment,
				RequestedTypes: types,
				Decided:        false,
			},
			AllowedFrom: []models.ApplicationStatus{models.STATUS_IN_REVIEW},
		}}
	}

	tests := []struct {
		name           string
		stored         []models.Application
		types          []models.DocumentType
		comment        string
		transitionErr  error
		want           *models.Application
		wantErr        error
		wantTransition []transitionCall
	}{
		{
			name:           "one type requested",
			stored:         []models.Application{inReview},
			types:          addressOnly,
			comment:        comment,
			want:           &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_DOCS_REQUESTED, Reason: comment, RequestedTypes: addressOnly},
			wantTransition: requestCall(addressOnly, comment),
		},
		{
			name:           "all types without comment",
			stored:         []models.Application{inReview},
			types:          allTypes,
			comment:        "",
			want:           &models.Application{ID: TEST_APPLICATION_ID, UserID: TEST_USER_ID, Status: models.STATUS_DOCS_REQUESTED, RequestedTypes: allTypes},
			wantTransition: requestCall(allTypes, ""),
		},
		{name: "nil types", stored: []models.Application{inReview}, types: nil, comment: comment, wantErr: domain.ErrDocumentTypesRequired},
		{name: "empty types", stored: []models.Application{inReview}, types: []models.DocumentType{}, comment: comment, wantErr: domain.ErrDocumentTypesRequired},
		{
			name:    "empty type among valid",
			stored:  []models.Application{inReview},
			types:   []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT, ""},
			comment: comment,
			wantErr: domain.ErrDocumentTypesRequired,
		},
		{
			name:    "unknown type among valid",
			stored:  []models.Application{inReview},
			types:   []models.DocumentType{models.DOCUMENT_TYPE_SELFIE, "driver_license"},
			comment: comment,
			wantErr: domain.ErrDocumentTypesRequired,
		},
		{
			name:    "only unknown type",
			stored:  []models.Application{inReview},
			types:   []models.DocumentType{"driver_license"},
			comment: comment,
			wantErr: domain.ErrDocumentTypesRequired,
		},
		{
			name:           "application not in review",
			stored:         []models.Application{submitted},
			types:          addressOnly,
			comment:        comment,
			wantErr:        domain.ErrStatusConflict,
			wantTransition: requestCall(addressOnly, comment),
		},
		{
			name:           "application not found",
			types:          addressOnly,
			comment:        comment,
			wantErr:        domain.ErrApplicationNotFound,
			wantTransition: requestCall(addressOnly, comment),
		},
		{
			name:           "transition fails",
			stored:         []models.Application{inReview},
			types:          addressOnly,
			comment:        comment,
			transitionErr:  errBoom,
			wantErr:        errBoom,
			wantTransition: requestCall(addressOnly, comment),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := newFakeApplicationRepository(tt.stored...)
			apps.transitionErr = tt.transitionErr
			customer := &fakeCustomerProfiles{}
			svc := newTestApplicationService(apps, customer, &fakeDocuments{})

			got, err := svc.RequestDocuments(t.Context(), TEST_APPLICATION_ID, officerID, tt.types, tt.comment)

			assert.Equal(t, tt.wantTransition, apps.transitionCalls)
			assert.Equal(t, []setStatusCall(nil), customer.setStatusCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
