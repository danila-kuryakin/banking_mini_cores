package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

type JWKSServer struct {
	server *http.Server
}

// NewJWKSServer для ротации ключей по стандартам RFC 7517 и OpenID Connect Discovery
func NewJWKSServer(addr string, jwks func() token.Set, logger *slog.Logger) *JWKSServer {
	mux := http.NewServeMux()
	mux.HandleFunc(domain.JWKS_PATH, func(w http.ResponseWriter, _ *http.Request) {
		body, err := json.Marshal(jwks())
		if err != nil {
			logger.Error("failed to marshal jwks", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set(domain.CONTENT_TYPE_HEADER, domain.JWKS_CONTENT_TYPE)
		_, _ = w.Write(body)
	})

	return &JWKSServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: domain.DEFAULT_READ_HEADER_TIMEOUT,
		},
	}
}

func (s *JWKSServer) Run() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve jwks on %s: %w", s.server.Addr, err)
	}

	return nil
}

func (s *JWKSServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *JWKSServer) Addr() string {
	return s.server.Addr
}
