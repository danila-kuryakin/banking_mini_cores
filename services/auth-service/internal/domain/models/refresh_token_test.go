package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const TEST_TOKEN_LIFETIME = time.Hour

// newTestRefreshToken отдаёт активный токен, истекающий через час после now.
func newTestRefreshToken(t *testing.T, now time.Time) RefreshToken {
	t.Helper()
	return RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: []byte("hash"),
		ExpiresAt: now.Add(TEST_TOKEN_LIFETIME),
		CreatedAt: now,
	}
}

func TestRefreshToken_IsActive_Active(t *testing.T) {
	t.Parallel()
	now := time.Now()

	assert.True(t, newTestRefreshToken(t, now).IsActive(now))
}

func TestRefreshToken_IsActive_Revoked(t *testing.T) {
	t.Parallel()
	now := time.Now()
	token := newTestRefreshToken(t, now)
	token.RevokedAt = &now

	assert.False(t, token.IsActive(now))
}

func TestRefreshToken_IsActive_Expired(t *testing.T) {
	t.Parallel()
	now := time.Now()
	token := newTestRefreshToken(t, now)

	assert.False(t, token.IsActive(now.Add(2*TEST_TOKEN_LIFETIME)))
}

func TestRefreshToken_IsActive_ExactlyAtExpiry(t *testing.T) {
	t.Parallel()
	now := time.Now()
	token := newTestRefreshToken(t, now)

	assert.False(t, token.IsActive(token.ExpiresAt))
}
