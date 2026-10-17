package server

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const TEST_FALLBACK_MESSAGE = "failed to do something"

func TestCredentialsError_Mapping(t *testing.T) {
	s := &AuthServer{log: slog.New(slog.DiscardHandler)}
	cases := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{name: "email required", err: domain.ErrEmailRequired, wantCode: codes.InvalidArgument, wantMsg: domain.ErrEmailRequired.Error()},
		{name: "password required", err: domain.ErrPasswordRequired, wantCode: codes.InvalidArgument, wantMsg: domain.ErrPasswordRequired.Error()},
		{name: "invalid credentials", err: domain.ErrInvalidCredentials, wantCode: codes.Unauthenticated, wantMsg: domain.ErrInvalidCredentials.Error()},
		{name: "wrapped invalid credentials", err: fmt.Errorf("wrap: %w", domain.ErrInvalidCredentials), wantCode: codes.Unauthenticated},
		{name: "unknown error", err: errors.New("db down"), wantCode: codes.Internal, wantMsg: TEST_FALLBACK_MESSAGE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.credentialsError(tc.err, TEST_FALLBACK_MESSAGE)

			require.Error(t, got)
			assert.Equal(t, tc.wantCode, status.Code(got))
			if tc.wantMsg != "" {
				assert.Equal(t, tc.wantMsg, status.Convert(got).Message())
			}
			assert.NotContains(t, status.Convert(got).Message(), "db down")
		})
	}

	t.Run("email taken", func(t *testing.T) {
		got := s.credentialsError(domain.ErrEmailTaken, TEST_FALLBACK_MESSAGE)

		require.ErrorIs(t, got, domain.ErrEmailAlreadyExists)
		assert.Equal(t, codes.AlreadyExists, status.Code(got))
	})
}

func TestRoleToProto_Mapping(t *testing.T) {
	cases := []struct {
		role models.Role
		want commonv1.Role
	}{
		{role: domain.ROLE_CLIENT, want: commonv1.Role_ROLE_CLIENT},
		{role: domain.ROLE_OFFICER, want: commonv1.Role_ROLE_OFFICER},
		{role: domain.ROLE_ADMIN, want: commonv1.Role_ROLE_ADMIN},
		{role: models.Role("auditor"), want: commonv1.Role_ROLE_UNSPECIFIED},
	}
	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			assert.Equal(t, tc.want, roleToProto(tc.role))
		})
	}
}
