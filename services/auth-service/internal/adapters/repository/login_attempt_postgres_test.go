package repository

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/migrations"
)

const (
	TEST_DB_PORT_ENV       = "AUTH_TEST_DB_PORT"
	TEST_DB_NAME           = "banking_auth"
	TEST_MAX_ATTEMPTS      = 5
	TEST_LOGIN_WINDOW      = 15 * time.Minute
	TEST_LOGIN_LOCK        = 15 * time.Minute
	TEST_SHORT_LOCK        = time.Minute
	TEST_EMAIL_DOMAIN      = "@example.com"
	TEST_ATTEMPTS_INTERVAL = time.Second
)

// testStart - явная точка отсчёта времени, чтобы тесты не зависели от часов.
var testStart = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

// testPool поднимает схему в отдельной базе; без AUTH_TEST_DB_PORT тест пропускается, чтобы не задеть рабочую базу.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	rawPort := os.Getenv(TEST_DB_PORT_ENV)
	if rawPort == "" {
		t.Skipf("%s is not set: the integration test needs a separate Postgres", TEST_DB_PORT_ENV)
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("%s: %v", TEST_DB_PORT_ENV, err)
	}

	cfg := config.DataBaseConfig{
		Host:     "localhost",
		Port:     port,
		User:     TEST_DB_NAME,
		Password: "123",
		DBName:   TEST_DB_NAME,
		SSLMode:  "disable",
		MaxConns: 5,
	}

	if err := migrator.Up(cfg, migrations.FS, migrations.Dir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := conn.NewConnectionDB(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

// testAttemptsRepo отдаёт репозиторий и уникальный email, строку которого удалит по окончании теста.
func testAttemptsRepo(t *testing.T) (*LoginAttemptRepo, string) {
	t.Helper()

	repo := NewLoginAttemptRepo(testPool(t))
	email := uuid.NewString() + TEST_EMAIL_DOMAIN
	t.Cleanup(func() { _ = repo.Reset(context.Background(), email) })

	return repo, email
}

// failureAt собирает неудачу в момент at по тестовым лимитам с блокировкой на lock.
func failureAt(at time.Time, lock time.Duration) models.LoginFailure {
	return models.LoginFailure{
		At:          at,
		WindowStart: at.Add(-TEST_LOGIN_WINDOW),
		MaxAttempts: TEST_MAX_ATTEMPTS,
		LockUntil:   at.Add(lock),
	}
}

// registerFailures пишет count неудач подряд с шагом в секунду начиная с from и отдаёт момент последней.
func registerFailures(t *testing.T, repo *LoginAttemptRepo, email string, from time.Time, count int, lock time.Duration) time.Time {
	t.Helper()

	at := from
	for i := 0; i < count; i++ {
		at = from.Add(time.Duration(i) * TEST_ATTEMPTS_INTERVAL)
		require.NoError(t, repo.RegisterFailure(context.Background(), email, failureAt(at, lock)))
	}

	return at
}

// requireNotLocked проверяет, что в момент now вход по email не заблокирован.
func requireNotLocked(t *testing.T, repo *LoginAttemptRepo, email string, now time.Time) {
	t.Helper()

	lockedUntil, err := repo.LockedUntil(context.Background(), email, now)
	require.NoError(t, err)
	require.Nil(t, lockedUntil)
}

func TestRegisterFailure_LocksAfterMaxAttempts(t *testing.T) {
	repo, email := testAttemptsRepo(t)

	last := registerFailures(t, repo, email, testStart, TEST_MAX_ATTEMPTS-1, TEST_LOGIN_LOCK)
	requireNotLocked(t, repo, email, last)

	at := last.Add(TEST_ATTEMPTS_INTERVAL)
	require.NoError(t, repo.RegisterFailure(context.Background(), email, failureAt(at, TEST_LOGIN_LOCK)))

	lockedUntil, err := repo.LockedUntil(context.Background(), email, at)
	require.NoError(t, err)
	require.NotNil(t, lockedUntil)
	require.True(t, lockedUntil.Equal(at.Add(TEST_LOGIN_LOCK)), "locked until mismatch: %s", lockedUntil)
}

func TestRegisterFailure_WindowExpiredRestartsCount(t *testing.T) {
	repo, email := testAttemptsRepo(t)

	registerFailures(t, repo, email, testStart, TEST_MAX_ATTEMPTS-1, TEST_LOGIN_LOCK)

	restart := testStart.Add(TEST_LOGIN_WINDOW + time.Minute)
	require.NoError(t, repo.RegisterFailure(context.Background(), email, failureAt(restart, TEST_LOGIN_LOCK)))
	requireNotLocked(t, repo, email, restart)

	last := registerFailures(t, repo, email, restart.Add(TEST_ATTEMPTS_INTERVAL), TEST_MAX_ATTEMPTS-2, TEST_LOGIN_LOCK)
	requireNotLocked(t, repo, email, last)
}

// Блокировка короче окна, чтобы счётчик сбросило именно истечение блокировки, а не окна.
func TestRegisterFailure_ExpiredLockRestartsCount(t *testing.T) {
	repo, email := testAttemptsRepo(t)

	last := registerFailures(t, repo, email, testStart, TEST_MAX_ATTEMPTS, TEST_SHORT_LOCK)
	lockedUntil, err := repo.LockedUntil(context.Background(), email, last)
	require.NoError(t, err)
	require.NotNil(t, lockedUntil)

	afterLock := lockedUntil.Add(TEST_ATTEMPTS_INTERVAL)
	require.NoError(t, repo.RegisterFailure(context.Background(), email, failureAt(afterLock, TEST_SHORT_LOCK)))
	requireNotLocked(t, repo, email, afterLock)

	next := registerFailures(t, repo, email, afterLock.Add(TEST_ATTEMPTS_INTERVAL), TEST_MAX_ATTEMPTS-2, TEST_SHORT_LOCK)
	requireNotLocked(t, repo, email, next)
}

func TestLockedUntil_ExpiredLock(t *testing.T) {
	repo, email := testAttemptsRepo(t)

	last := registerFailures(t, repo, email, testStart, TEST_MAX_ATTEMPTS, TEST_LOGIN_LOCK)

	requireNotLocked(t, repo, email, last.Add(TEST_LOGIN_LOCK))
}

func TestReset_DeletesAttempts(t *testing.T) {
	repo, email := testAttemptsRepo(t)

	registerFailures(t, repo, email, testStart, TEST_MAX_ATTEMPTS-1, TEST_LOGIN_LOCK)
	require.NoError(t, repo.Reset(context.Background(), email))

	last := registerFailures(t, repo, email, testStart.Add(time.Minute), TEST_MAX_ATTEMPTS-1, TEST_LOGIN_LOCK)
	requireNotLocked(t, repo, email, last)

	require.NoError(t, repo.Reset(context.Background(), uuid.NewString()+TEST_EMAIL_DOMAIN))
}
