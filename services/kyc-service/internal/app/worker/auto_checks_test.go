package worker

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

const (
	TEST_FIRST_NAME          = "Ivan"
	TEST_LAST_NAME           = "Petrov"
	TEST_SANCTION_FIRST_NAME = "IVAN"
	TEST_SANCTION_LAST_NAME  = "PETROV"
	TEST_SANCTION_SOURCE     = "OFAC"
	TEST_AUTO_BATCH_SIZE     = 3
	TEST_INTERVAL            = time.Minute
	TEST_CHECK_SANCTIONS     = models.CHECK_TYPE_SANCTIONS
	TEST_CHECK_AGE           = models.CHECK_TYPE_AGE
	TEST_RESULT_PASSED       = models.CHECK_RESULT_PASSED
	TEST_RESULT_FAILED       = models.CHECK_RESULT_FAILED
	TEST_DETAILS_NO_MATCH    = "no match"
	TEST_DETAILS_MATCH       = "match in OFAC: IVAN PETROV"
	TEST_DETAILS_UNDERAGE    = "age 17 is under 18"
	TEST_DETAILS_NO_BIRTH    = "birth date is missing"
	TEST_REASON_BOTH         = "kyc auto check failed: sanctions: match in OFAC: IVAN PETROV; age: age 17 is under 18"
	TEST_REASON_SANCTIONS    = "kyc auto check failed: sanctions: match in OFAC: IVAN PETROV"
	TEST_REASON_UNDERAGE     = "kyc auto check failed: age: age 17 is under 18"
	TEST_REASON_NO_BIRTH     = "kyc auto check failed: age: birth date is missing"
)

var (
	TEST_APPLICATION_ID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_APPLICATION_ID_2 = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TEST_APPLICATION_ID_3 = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	TEST_USER_ID          = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	TEST_USER_ID_2        = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	TEST_USER_ID_3        = uuid.MustParse("77777777-7777-7777-7777-777777777777")
	TEST_ERR_FIND_MATCH   = errors.New("find match failed")
	TEST_ERR_SAVE_CHECK   = errors.New("save check failed")
	TEST_ERR_DB           = errors.New("database is down")
	TEST_ERR_SET_STATUS   = errors.New("set status failed")
)

// autoChecksDeps - фейки зависимостей AutoChecks
type autoChecksDeps struct {
	apps      *fakeApplicationRepository
	checks    *fakeCheckRepository
	sanctions *fakeSanctionsRepository
	customer  *fakeCustomerProfiles
}

// newAutoChecksDeps - пустые фейки зависимостей
func newAutoChecksDeps() autoChecksDeps {
	return autoChecksDeps{
		apps:      &fakeApplicationRepository{},
		checks:    &fakeCheckRepository{},
		sanctions: &fakeSanctionsRepository{},
		customer:  &fakeCustomerProfiles{},
	}
}

// newTestAutoChecks - воркер поверх фейков
func newTestAutoChecks(deps autoChecksDeps, batchSize int) *AutoChecks {
	repo := &repository.Repository{Application: deps.apps, Check: deps.checks, Sanctions: deps.sanctions}
	clients := &client.Clients{Customer: deps.customer}
	return NewAutoChecks(repo, clients, slog.New(slog.DiscardHandler), TEST_INTERVAL, batchSize)
}

// testToday - сегодняшняя полночь UTC; 29 февраля заменяется на 28-е, чтобы AddDate не нормализовал дату
func testToday() time.Time {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if today.Month() == time.February && today.Day() == 29 {
		return today.AddDate(0, 0, -1)
	}
	return today
}

// birthDateAgo - дата рождения ровно years лет назад
func birthDateAgo(years int) *time.Time {
	birthDate := testToday().AddDate(-years, 0, 0)
	return &birthDate
}

// testApplication - заявка со снимком анкеты
func testApplication(applicationID, userID uuid.UUID, birthDate *time.Time) *models.Application {
	return &models.Application{
		ID:        applicationID,
		UserID:    userID,
		FirstName: TEST_FIRST_NAME,
		LastName:  TEST_LAST_NAME,
		BirthDate: birthDate,
	}
}

// testSanctionEntry - запись санкционного списка, совпадающая с заявителем
func testSanctionEntry() *models.SanctionEntry {
	return &models.SanctionEntry{
		ID:        uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		FirstName: TEST_SANCTION_FIRST_NAME,
		LastName:  TEST_SANCTION_LAST_NAME,
		BirthDate: time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC),
		Source:    TEST_SANCTION_SOURCE,
	}
}

// check - ожидаемый результат проверки, сохраняемый через SaveCheck
func check(applicationID uuid.UUID, checkType models.CheckType, result models.CheckResult, details string) models.Check {
	return models.Check{ApplicationID: applicationID, Type: checkType, Result: result, Details: details}
}

// toInReview - ожидаемый вызов Transition в in_review
func toInReview(applicationID uuid.UUID) transitionCall {
	return transitionCall{
		applicationID: applicationID,
		transition:    models.Transition{To: models.STATUS_IN_REVIEW},
		allowedFrom:   []models.ApplicationStatus{models.STATUS_AUTO_CHECKS},
	}
}

// toAutoRejected - ожидаемый вызов Transition в auto_rejected с причиной
func toAutoRejected(applicationID uuid.UUID, reason string) transitionCall {
	return transitionCall{
		applicationID: applicationID,
		transition:    models.Transition{To: models.STATUS_AUTO_REJECTED, Comment: reason, Reason: reason, Decided: true},
		allowedFrom:   []models.ApplicationStatus{models.STATUS_AUTO_CHECKS},
	}
}

// toProfileFilled - ожидаемый вызов SetStatus с возвратом клиента в profile_filled
func toProfileFilled(userID uuid.UUID, reason string) setStatusCall {
	return setStatusCall{userID: userID, to: models.CUSTOMER_STATUS_PROFILE_FILLED, reason: reason, actorID: nil}
}

func TestAutoChecks_process(t *testing.T) {
	tests := []struct {
		name            string
		app             *models.Application
		match           *models.SanctionEntry
		findMatchErr    error
		saveErr         error
		saveErrOnCall   int
		transitionErr   error
		setStatusErr    error
		wantErr         error
		wantSaves       []models.Check
		wantTransitions []transitionCall
		wantSetStatuses []setStatusCall
	}{
		{
			name: "all checks passed - in_review",
			app:  testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
			},
			wantTransitions: []transitionCall{toInReview(TEST_APPLICATION_ID)},
		},
		{
			name: "exactly 18 years - in_review",
			app:  testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(18)),
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 18"),
			},
			wantTransitions: []transitionCall{toInReview(TEST_APPLICATION_ID)},
		},
		{
			name:  "sanctions match and underage - auto_rejected with both reasons",
			app:   testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			match: testSanctionEntry(),
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_FAILED, TEST_DETAILS_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
			},
			wantTransitions: []transitionCall{toAutoRejected(TEST_APPLICATION_ID, TEST_REASON_BOTH)},
			wantSetStatuses: []setStatusCall{toProfileFilled(TEST_USER_ID, TEST_REASON_BOTH)},
		},
		{
			name: "birth date missing - auto_rejected by age only",
			app:  testApplication(TEST_APPLICATION_ID, TEST_USER_ID, nil),
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_NO_BIRTH),
			},
			wantTransitions: []transitionCall{toAutoRejected(TEST_APPLICATION_ID, TEST_REASON_NO_BIRTH)},
			wantSetStatuses: []setStatusCall{toProfileFilled(TEST_USER_ID, TEST_REASON_NO_BIRTH)},
		},
		{
			name:  "adult in sanctions list - auto_rejected by sanctions only",
			app:   testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
			match: testSanctionEntry(),
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_FAILED, TEST_DETAILS_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
			},
			wantTransitions: []transitionCall{toAutoRejected(TEST_APPLICATION_ID, TEST_REASON_SANCTIONS)},
			wantSetStatuses: []setStatusCall{toProfileFilled(TEST_USER_ID, TEST_REASON_SANCTIONS)},
		},
		{
			name:         "find match error - nothing saved",
			app:          testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
			findMatchErr: TEST_ERR_FIND_MATCH,
			wantErr:      TEST_ERR_FIND_MATCH,
		},
		{
			name:          "first save check error - stops",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
			saveErr:       TEST_ERR_SAVE_CHECK,
			saveErrOnCall: 1,
			wantErr:       TEST_ERR_SAVE_CHECK,
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
			},
		},
		{
			name:          "second save check error - no transition",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			saveErr:       TEST_ERR_SAVE_CHECK,
			saveErrOnCall: 2,
			wantErr:       TEST_ERR_SAVE_CHECK,
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
			},
		},
		{
			name:          "transition to in_review conflict - returned",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
			transitionErr: domain.ErrStatusConflict,
			wantErr:       domain.ErrStatusConflict,
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
			},
			wantTransitions: []transitionCall{toInReview(TEST_APPLICATION_ID)},
		},
		{
			name:          "transition to auto_rejected not found - no set status",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			transitionErr: domain.ErrApplicationNotFound,
			wantErr:       domain.ErrApplicationNotFound,
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
			},
			wantTransitions: []transitionCall{toAutoRejected(TEST_APPLICATION_ID, TEST_REASON_UNDERAGE)},
		},
		{
			name:          "transition to auto_rejected db error - no set status",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			transitionErr: TEST_ERR_DB,
			wantErr:       TEST_ERR_DB,
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
			},
			wantTransitions: []transitionCall{toAutoRejected(TEST_APPLICATION_ID, TEST_REASON_UNDERAGE)},
		},
		{
			name:         "set status error after auto_rejected - returned",
			app:          testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			setStatusErr: TEST_ERR_SET_STATUS,
			wantErr:      TEST_ERR_SET_STATUS,
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
			},
			wantTransitions: []transitionCall{toAutoRejected(TEST_APPLICATION_ID, TEST_REASON_UNDERAGE)},
			wantSetStatuses: []setStatusCall{toProfileFilled(TEST_USER_ID, TEST_REASON_UNDERAGE)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newAutoChecksDeps()
			deps.sanctions.match = tt.match
			deps.sanctions.findMatchErr = tt.findMatchErr
			deps.checks.saveErr = tt.saveErr
			deps.checks.saveErrOnCall = tt.saveErrOnCall
			deps.apps.transitionErr = tt.transitionErr
			deps.customer.setStatusErr = tt.setStatusErr
			w := newTestAutoChecks(deps, TEST_AUTO_BATCH_SIZE)

			err := w.process(t.Context(), tt.app)

			assert.Equal(t, tt.wantSaves, deps.checks.saves())
			assert.Equal(t, tt.wantTransitions, deps.apps.transitions())
			assert.Equal(t, tt.wantSetStatuses, deps.customer.setStatuses())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// Инвариант: SetStatus вызывается тогда и только тогда, когда заявка успешно ушла в auto_rejected
func TestAutoChecks_process_SetStatusOnlyAfterAutoReject(t *testing.T) {
	tests := []struct {
		name            string
		app             *models.Application
		findMatchErr    error
		saveErr         error
		saveErrOnCall   int
		transitionErr   error
		wantErr         error
		wantSetStatuses []setStatusCall
	}{
		{
			name: "all passed",
			app:  testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
		},
		{
			name:            "auto_rejected successfully",
			app:             testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			wantSetStatuses: []setStatusCall{toProfileFilled(TEST_USER_ID, TEST_REASON_UNDERAGE)},
		},
		{
			name:          "auto_rejected transition failed",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			transitionErr: TEST_ERR_DB,
			wantErr:       TEST_ERR_DB,
		},
		{
			name:         "checks failed with error",
			app:          testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			findMatchErr: TEST_ERR_FIND_MATCH,
			wantErr:      TEST_ERR_FIND_MATCH,
		},
		{
			name:          "rejected but second save failed",
			app:           testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(17)),
			saveErr:       TEST_ERR_SAVE_CHECK,
			saveErrOnCall: 2,
			wantErr:       TEST_ERR_SAVE_CHECK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newAutoChecksDeps()
			deps.sanctions.findMatchErr = tt.findMatchErr
			deps.checks.saveErr = tt.saveErr
			deps.checks.saveErrOnCall = tt.saveErrOnCall
			deps.apps.transitionErr = tt.transitionErr
			w := newTestAutoChecks(deps, TEST_AUTO_BATCH_SIZE)

			err := w.process(t.Context(), tt.app)

			assert.Equal(t, tt.wantSetStatuses, deps.customer.setStatuses())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAutoChecks_processBatch(t *testing.T) {
	tests := []struct {
		name               string
		batchSize          int
		takeResult         []*models.Application
		takeErr            error
		transitionErrByID  map[uuid.UUID]error
		setStatusErrByUser map[uuid.UUID]error
		wantTakes          []int
		wantSaves          []models.Check
		wantTransitions    []transitionCall
		wantSetStatuses    []setStatusCall
	}{
		{
			name:      "two applications processed in order",
			batchSize: TEST_AUTO_BATCH_SIZE,
			takeResult: []*models.Application{
				testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
				testApplication(TEST_APPLICATION_ID_2, TEST_USER_ID_2, birthDateAgo(40)),
			},
			wantTakes: []int{TEST_AUTO_BATCH_SIZE},
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
				check(TEST_APPLICATION_ID_2, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID_2, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 40"),
			},
			wantTransitions: []transitionCall{toInReview(TEST_APPLICATION_ID), toInReview(TEST_APPLICATION_ID_2)},
		},
		{
			name:      "second transition fails - third still processed",
			batchSize: TEST_AUTO_BATCH_SIZE,
			takeResult: []*models.Application{
				testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
				testApplication(TEST_APPLICATION_ID_2, TEST_USER_ID_2, birthDateAgo(30)),
				testApplication(TEST_APPLICATION_ID_3, TEST_USER_ID_3, birthDateAgo(30)),
			},
			transitionErrByID: map[uuid.UUID]error{TEST_APPLICATION_ID_2: TEST_ERR_DB},
			wantTakes:         []int{TEST_AUTO_BATCH_SIZE},
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
				check(TEST_APPLICATION_ID_2, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID_2, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
				check(TEST_APPLICATION_ID_3, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID_3, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
			},
			wantTransitions: []transitionCall{
				toInReview(TEST_APPLICATION_ID),
				toInReview(TEST_APPLICATION_ID_2),
				toInReview(TEST_APPLICATION_ID_3),
			},
		},
		{
			name:      "second set status fails - third still auto_rejected",
			batchSize: TEST_AUTO_BATCH_SIZE,
			takeResult: []*models.Application{
				testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
				testApplication(TEST_APPLICATION_ID_2, TEST_USER_ID_2, birthDateAgo(17)),
				testApplication(TEST_APPLICATION_ID_3, TEST_USER_ID_3, birthDateAgo(17)),
			},
			setStatusErrByUser: map[uuid.UUID]error{TEST_USER_ID_2: TEST_ERR_SET_STATUS},
			wantTakes:          []int{TEST_AUTO_BATCH_SIZE},
			wantSaves: []models.Check{
				check(TEST_APPLICATION_ID, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID, TEST_CHECK_AGE, TEST_RESULT_PASSED, "age 30"),
				check(TEST_APPLICATION_ID_2, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID_2, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
				check(TEST_APPLICATION_ID_3, TEST_CHECK_SANCTIONS, TEST_RESULT_PASSED, TEST_DETAILS_NO_MATCH),
				check(TEST_APPLICATION_ID_3, TEST_CHECK_AGE, TEST_RESULT_FAILED, TEST_DETAILS_UNDERAGE),
			},
			wantTransitions: []transitionCall{
				toInReview(TEST_APPLICATION_ID),
				toAutoRejected(TEST_APPLICATION_ID_2, TEST_REASON_UNDERAGE),
				toAutoRejected(TEST_APPLICATION_ID_3, TEST_REASON_UNDERAGE),
			},
			wantSetStatuses: []setStatusCall{
				toProfileFilled(TEST_USER_ID_2, TEST_REASON_UNDERAGE),
				toProfileFilled(TEST_USER_ID_3, TEST_REASON_UNDERAGE),
			},
		},
		{
			name:       "empty batch - only take",
			batchSize:  TEST_AUTO_BATCH_SIZE,
			takeResult: []*models.Application{},
			wantTakes:  []int{TEST_AUTO_BATCH_SIZE},
		},
		{
			name:      "nil batch - only take",
			batchSize: TEST_AUTO_BATCH_SIZE,
			wantTakes: []int{TEST_AUTO_BATCH_SIZE},
		},
		{
			name:      "take error - nothing processed",
			batchSize: TEST_AUTO_BATCH_SIZE,
			takeResult: []*models.Application{
				testApplication(TEST_APPLICATION_ID, TEST_USER_ID, birthDateAgo(30)),
			},
			takeErr:   TEST_ERR_DB,
			wantTakes: []int{TEST_AUTO_BATCH_SIZE},
		},
		{
			name:       "batch size from constructor",
			batchSize:  1,
			takeResult: []*models.Application{},
			wantTakes:  []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newAutoChecksDeps()
			deps.apps.takeResult = tt.takeResult
			deps.apps.takeErr = tt.takeErr
			deps.apps.transitionErrByID = tt.transitionErrByID
			deps.customer.setStatusErrByUser = tt.setStatusErrByUser
			w := newTestAutoChecks(deps, tt.batchSize)

			w.processBatch(t.Context())

			assert.Equal(t, tt.wantTakes, deps.apps.takes())
			assert.Equal(t, tt.wantSaves, deps.checks.saves())
			assert.Equal(t, tt.wantTransitions, deps.apps.transitions())
			assert.Equal(t, tt.wantSetStatuses, deps.customer.setStatuses())
		})
	}
}

// requireDone - Run уже вернулся
func requireDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	default:
		t.Fatalf("Run did not return after context cancellation")
	}
}

// requireRunning - Run ещё работает
func requireRunning(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		t.Fatalf("Run returned before context cancellation")
	default:
	}
}

// startAutoChecks - запускает Run в горутине; done закрывается при возврате
func startAutoChecks(ctx context.Context, w *AutoChecks) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()
	return done
}

func TestAutoChecks_Run_TicksAndStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deps := newAutoChecksDeps()
		deps.apps.takeResult = []*models.Application{}
		w := newTestAutoChecks(deps, TEST_AUTO_BATCH_SIZE)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := startAutoChecks(ctx, w)

		time.Sleep(TEST_INTERVAL - time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, []int(nil), deps.apps.takes())

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, []int{TEST_AUTO_BATCH_SIZE}, deps.apps.takes())

		time.Sleep(2 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_AUTO_BATCH_SIZE, TEST_AUTO_BATCH_SIZE, TEST_AUTO_BATCH_SIZE}, deps.apps.takes())
		requireRunning(t, done)

		cancel()
		synctest.Wait()
		requireDone(t, done)

		time.Sleep(2 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_AUTO_BATCH_SIZE, TEST_AUTO_BATCH_SIZE, TEST_AUTO_BATCH_SIZE}, deps.apps.takes())
	})
}

func TestAutoChecks_Run_ContinuesAfterBatchError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deps := newAutoChecksDeps()
		deps.apps.takeErr = TEST_ERR_DB
		w := newTestAutoChecks(deps, TEST_AUTO_BATCH_SIZE)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := startAutoChecks(ctx, w)

		time.Sleep(3 * TEST_INTERVAL)
		synctest.Wait()
		assert.Equal(t, []int{TEST_AUTO_BATCH_SIZE, TEST_AUTO_BATCH_SIZE, TEST_AUTO_BATCH_SIZE}, deps.apps.takes())
		requireRunning(t, done)

		cancel()
		synctest.Wait()
		requireDone(t, done)
	})
}

func TestAutoChecks_Run_CanceledBeforeStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deps := newAutoChecksDeps()
		deps.apps.takeResult = []*models.Application{}
		w := newTestAutoChecks(deps, TEST_AUTO_BATCH_SIZE)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		done := startAutoChecks(ctx, w)

		synctest.Wait()
		requireDone(t, done)
		assert.Equal(t, []int(nil), deps.apps.takes())
	})
}
