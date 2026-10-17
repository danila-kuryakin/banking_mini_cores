package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TEST_UNKNOWN_PATH = "/jwks.json"

// stubJWKS отдаёт фиксированный набор из двух ключей.
func stubJWKS() token.Set {
	return token.Set{Keys: []token.JWK{
		{KeyType: "RSA", Use: "sig", Algorithm: "RS256", KeyID: "kid-1", Modulus: "n-1", Exponent: "AQAB"},
		{KeyType: "RSA", Use: "sig", Algorithm: "RS256", KeyID: "kid-2", Modulus: "n-2", Exponent: "AQAB"},
	}}
}

// serve прогоняет запрос через хендлер сервера без поднятия сокета.
func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	s := NewJWKSServer(":0", stubJWKS, slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestJWKSHandler_ServesKeys(t *testing.T) {
	t.Parallel()

	rec := serve(t, domain.JWKS_PATH)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, domain.JWKS_CONTENT_TYPE, rec.Header().Get(domain.CONTENT_TYPE_HEADER))
	var got token.Set
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, stubJWKS(), got)
}

func TestJWKSHandler_UnknownPath(t *testing.T) {
	t.Parallel()

	rec := serve(t, TEST_UNKNOWN_PATH)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

const (
	TEST_LOOPBACK_ADDR  = "127.0.0.1:0"
	TEST_WAIT_TIMEOUT   = 2 * time.Second
	TEST_WAIT_TICK      = 20 * time.Millisecond
	TEST_RUN_ERR_PREFIX = "serve jwks on"
)

// freeAddr отдаёт адрес свободного порта на 127.0.0.1.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", TEST_LOOPBACK_ADDR)
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

func TestJWKSServer_RunPortBusy(t *testing.T) {
	l, err := net.Listen("tcp", TEST_LOOPBACK_ADDR)
	require.NoError(t, err)
	defer l.Close()

	err = NewJWKSServer(l.Addr().String(), stubJWKS, slog.New(slog.DiscardHandler)).Run()

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), TEST_RUN_ERR_PREFIX), "unexpected error text: %v", err)
}

func TestJWKSServer_ShutdownReturnsNil(t *testing.T) {
	addr := freeAddr(t)
	s := NewJWKSServer(addr, stubJWKS, slog.New(slog.DiscardHandler))
	runErr := make(chan error, 1)
	go func() { runErr <- s.Run() }()

	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, TEST_WAIT_TIMEOUT, TEST_WAIT_TICK, "jwks server did not start accepting connections")

	ctx, cancel := context.WithTimeout(context.Background(), TEST_WAIT_TIMEOUT)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))

	select {
	case err := <-runErr:
		require.NoError(t, err)
	case <-time.After(TEST_WAIT_TIMEOUT):
		t.Fatalf("run did not return after shutdown")
	}
}
