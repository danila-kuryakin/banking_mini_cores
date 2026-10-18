package password

import (
	"context"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_SHORT_WAIT      = 50 * time.Millisecond
	TEST_LONG_WAIT       = time.Second
	TEST_CANCEL_ATTEMPTS = 20
)

// occupy занимает место в семафоре и освобождает его по окончании теста.
func occupy(t *testing.T, h *Hasher) {
	t.Helper()
	release, err := h.acquire(context.Background())
	require.NoError(t, err)
	t.Cleanup(release)
}

func TestHasher_AcquireWithinLimit(t *testing.T) {
	t.Parallel()
	h := NewHasher(2, TEST_LONG_WAIT)

	first, err := h.acquire(context.Background())
	require.NoError(t, err)
	defer first()
	second, err := h.acquire(context.Background())
	require.NoError(t, err)
	defer second()
}

func TestHasher_AcquireBusy(t *testing.T) {
	t.Parallel()
	h := NewHasher(1, TEST_SHORT_WAIT)
	occupy(t, h)

	_, err := h.acquire(context.Background())

	require.ErrorIs(t, err, domain.ErrPasswordHasherBusy)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestHasher_AcquireAfterRelease(t *testing.T) {
	t.Parallel()
	h := NewHasher(1, TEST_LONG_WAIT)
	release, err := h.acquire(context.Background())
	require.NoError(t, err)
	result := make(chan error, 1)

	go func() {
		next, err := h.acquire(context.Background())
		if err == nil {
			next()
		}
		result <- err
	}()
	release()

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(TEST_LONG_WAIT):
		t.Fatalf("acquire did not return after release")
	}
}

func TestHasher_CanceledContext(t *testing.T) {
	t.Parallel()
	h := NewHasher(1, TEST_LONG_WAIT)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for i := 0; i < TEST_CANCEL_ATTEMPTS; i++ {
		release, err := h.acquire(ctx)

		require.ErrorIs(t, err, domain.ErrPasswordHasherBusy)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, release)
	}
}

func TestHasher_MaxWaitShorterThanContext(t *testing.T) {
	t.Parallel()
	h := NewHasher(1, TEST_SHORT_WAIT)
	occupy(t, h)
	result := make(chan error, 1)

	go func() {
		_, err := h.acquire(context.Background())
		result <- err
	}()

	select {
	case err := <-result:
		require.ErrorIs(t, err, domain.ErrPasswordHasherBusy)
	case <-time.After(TEST_LONG_WAIT):
		t.Fatalf("acquire waited longer than max wait")
	}
}

func TestHasher_InvalidLimit(t *testing.T) {
	t.Parallel()
	h := NewHasher(0, TEST_SHORT_WAIT)
	occupy(t, h)

	_, err := h.acquire(context.Background())

	require.ErrorIs(t, err, domain.ErrPasswordHasherBusy)
}

func TestHasher_HashVerifyRoundTrip(t *testing.T) {
	h := NewHasher(1, TEST_LONG_WAIT)

	encoded, err := h.Hash(context.Background(), TEST_PASSWORD)
	require.NoError(t, err)

	assert.NoError(t, h.Verify(context.Background(), encoded, TEST_PASSWORD))
}

func TestHasher_VerifyBusy(t *testing.T) {
	encoded, err := Hash(TEST_PASSWORD)
	require.NoError(t, err)
	h := NewHasher(1, TEST_SHORT_WAIT)
	occupy(t, h)

	err = h.Verify(context.Background(), encoded, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrPasswordHasherBusy)
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
}
