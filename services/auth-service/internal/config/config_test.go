package config

import (
	"testing"

	"github.com/stretchr/testify/require"
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
