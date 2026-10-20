package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

const TEST_CUSTOMER_UPSTREAM = "localhost:50052"

func TestValidate_EmptyCustomerUpstream(t *testing.T) {
	t.Parallel()

	require.Error(t, (&Config{}).Validate())
}

func TestValidate_Valid(t *testing.T) {
	t.Parallel()

	cfg := &Config{Upstreams: Upstreams{Customer: TEST_CUSTOMER_UPSTREAM}}

	require.NoError(t, cfg.Validate())
}

func TestValidate_NegativeLoginLimits(t *testing.T) {
	t.Parallel()

	cases := map[string]Login{
		"max failed attempts": {MaxFailedAttempts: -1},
		"window":              {Window: -time.Minute},
		"lock duration":       {LockDuration: -time.Minute},
	}
	for name, login := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{Upstreams: Upstreams{Customer: TEST_CUSTOMER_UPSTREAM}, Login: login}

			require.Error(t, cfg.Validate())
		})
	}
}

func TestValidate_ZeroLoginLimits(t *testing.T) {
	t.Parallel()

	cfg := &Config{Upstreams: Upstreams{Customer: TEST_CUSTOMER_UPSTREAM}, Login: Login{}}

	require.NoError(t, cfg.Validate())
}

func TestValidate_NegativePasswordHashing(t *testing.T) {
	t.Parallel()

	cases := map[string]PasswordHashing{
		"max concurrent": {MaxConcurrent: -1},
		"max wait":       {MaxWait: -time.Second},
	}
	for name, hashing := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{Upstreams: Upstreams{Customer: TEST_CUSTOMER_UPSTREAM}, PasswordHashing: hashing}

			require.Error(t, cfg.Validate())
		})
	}
}

func TestValidate_ZeroPasswordHashing(t *testing.T) {
	t.Parallel()

	cfg := &Config{Upstreams: Upstreams{Customer: TEST_CUSTOMER_UPSTREAM}, PasswordHashing: PasswordHashing{}}

	require.NoError(t, cfg.Validate())
}

// configWithAdminPassword отдаёт валидный конфиг с заданным паролем админа.
func configWithAdminPassword(t *testing.T, adminPassword string) *Config {
	t.Helper()
	return &Config{Upstreams: Upstreams{Customer: TEST_CUSTOMER_UPSTREAM}, Admin: Admin{Password: adminPassword}}
}

func TestValidate_AdminPasswordEmpty(t *testing.T) {
	t.Parallel()

	require.NoError(t, configWithAdminPassword(t, "").Validate())
}

func TestValidate_AdminPasswordTooShort(t *testing.T) {
	t.Parallel()

	require.Error(t, configWithAdminPassword(t, strings.Repeat("p", domain.PASSWORD_MIN_RUNES-1)).Validate())
}

func TestValidate_AdminPasswordTooLong(t *testing.T) {
	t.Parallel()

	require.Error(t, configWithAdminPassword(t, strings.Repeat("p", domain.PASSWORD_MAX_RUNES+1)).Validate())
}

func TestValidate_AdminPasswordBounds(t *testing.T) {
	t.Parallel()

	require.NoError(t, configWithAdminPassword(t, strings.Repeat("p", domain.PASSWORD_MIN_RUNES)).Validate())
	require.NoError(t, configWithAdminPassword(t, strings.Repeat("p", domain.PASSWORD_MAX_RUNES)).Validate())
}

func TestValidate_AdminPasswordCountsRunes(t *testing.T) {
	t.Parallel()

	require.NoError(t, configWithAdminPassword(t, strings.Repeat("ж", domain.PASSWORD_MIN_RUNES)).Validate())
	require.Error(t, configWithAdminPassword(t, strings.Repeat("ж", domain.PASSWORD_MIN_RUNES-1)).Validate())
}
