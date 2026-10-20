package server

import (
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	platformconfig "github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_HOST            = "127.0.0.1"
	TEST_FREE_PORT       = "0"
	TEST_REQUEST_TIMEOUT = 5 * time.Second
	TEST_RUN_WAIT        = 5 * time.Second
	TEST_ACCESS_TTL      = 15 * time.Minute
	TEST_REFRESH_TTL     = 720 * time.Hour
)

func TestServer_RunFailsWhenJWKSFails(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	busy, err := net.Listen("tcp", net.JoinHostPort(TEST_HOST, TEST_FREE_PORT))
	require.NoError(t, err)
	defer busy.Close()
	busyPort := strconv.Itoa(busy.Addr().(*net.TCPAddr).Port)

	manager, err := token.NewManager("", TEST_ACCESS_TTL, TEST_REFRESH_TTL, logger)
	require.NoError(t, err)
	cfg := &config.Config{
		Server:          platformconfig.Server{Host: TEST_HOST, Port: TEST_FREE_PORT},
		JWKS:            platformconfig.Server{Host: TEST_HOST, Port: busyPort},
		RequestTimeout:  TEST_REQUEST_TIMEOUT,
		PasswordHashing: config.PasswordHashing{MaxConcurrent: 4, MaxWait: time.Second},
	}
	svc := service.NewService(&repository.Repository{Auth: &mocks.AuthRepositoryMock{}}, manager, &mocks.CustomerProfilesMock{}, cfg)

	runErr := make(chan error, 1)
	// gRPC-сервер после теста остаётся работать до конца процесса - так задумано.
	go func() { runErr <- NewServer(cfg, svc, manager.JWKS, logger).Run() }()

	select {
	case err := <-runErr:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwks server")
	case <-time.After(TEST_RUN_WAIT):
		t.Fatalf("run did not return after jwks failure")
	}
}
