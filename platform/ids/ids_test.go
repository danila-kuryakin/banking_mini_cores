package ids_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking-kyc/platform/ids"
)

func TestNewProducesVersion7(t *testing.T) {
	t.Parallel()

	id := ids.New()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("uuid.Parse(%q): %v", id, err)
	}

	if got, want := parsed.Version(), uuid.Version(7); got != want {
		t.Errorf("version = %d, want %d", got, want)
	}

	if got, want := parsed.Variant(), uuid.RFC4122; got != want {
		t.Errorf("variant = %v, want %v", got, want)
	}
}

// Chronological sortability is the reason for choosing v7 over v4: it is what
// keeps Postgres index inserts appending rather than scattering.
func TestIdsSortChronologically(t *testing.T) {
	t.Parallel()

	const n = 200

	previous := ids.New()

	for range n {
		current := ids.New()

		if current < previous {
			t.Fatalf("ids went backwards: %q then %q", previous, current)
		}

		previous = current
	}
}

func TestNewIsUnique(t *testing.T) {
	t.Parallel()

	const n = 10_000

	seen := make(map[string]struct{}, n)

	for range n {
		id := ids.New()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id generated: %q", id)
		}

		seen[id] = struct{}{}
	}
}

func TestParseRejectsWrongVersion(t *testing.T) {
	t.Parallel()

	v4 := uuid.New().String()

	if _, err := ids.Parse(v4); !errors.Is(err, ids.ErrNotV7) {
		t.Errorf("Parse(v4) error = %v, want ErrNotV7", err)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"", "not-a-uuid", "12345", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"} {
		if ids.IsValid(s) {
			t.Errorf("IsValid(%q) = true, want false", s)
		}
	}
}

func TestParseCanonicalises(t *testing.T) {
	t.Parallel()

	id := ids.New()

	got, err := ids.Parse(id)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got != id {
		t.Errorf("Parse(%q) = %q, want the canonical form unchanged", id, got)
	}
}
