package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
