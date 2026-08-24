// Package ids generates and validates the identifiers used across the system.
//
// Every identifier is a UUID v7: it embeds a millisecond timestamp in its
// leading bits, so ids sort chronologically. That keeps Postgres B-tree
// inserts appending to the right-hand edge of the index instead of scattering
// across it the way UUID v4 does, and it makes an id self-describing in a log
// line.
package ids

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrNotV7 is returned for a syntactically valid UUID of the wrong version.
var ErrNotV7 = errors.New("ids: identifier is not a UUID v7")

// New returns a fresh UUID v7 as a canonical lowercase string.
//
// uuid.NewV7 draws from crypto/rand and only fails if the entropy source does,
// which on a running process means something is deeply wrong. Callers should
// not have to handle that on every insert, so it is a panic rather than an
// error; use NewE if you need to decide for yourself.
func New() string {
	id, err := NewE()
	if err != nil {
		panic(fmt.Sprintf("ids: cannot generate UUID v7: %v", err))
	}

	return id
}

// NewE is New with the entropy failure surfaced as an error.
func NewE() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuid v7: %w", err)
	}

	return id.String(), nil
}

// Parse validates that s is a UUID v7 and returns it in canonical form. Use it
// at the edges, on request fields, so that a malformed id fails with
// INVALID_ARGUMENT instead of turning into a confusing NOT_FOUND.
func Parse(s string) (string, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return "", fmt.Errorf("parse uuid: %w", err)
	}

	if id.Version() != 7 {
		return "", fmt.Errorf("%w: got version %d", ErrNotV7, id.Version())
	}

	return id.String(), nil
}

// IsValid reports whether s is a UUID v7.
func IsValid(s string) bool {
	_, err := Parse(s)

	return err == nil
}
