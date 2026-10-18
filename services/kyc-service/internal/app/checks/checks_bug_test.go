//go:build !gremlins

package checks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFullYears_LeapBirthNonLeapYear - R9: рождённому 29.02 год засчитывается 28.02 невисокосного года
func TestFullYears_LeapBirthNonLeapYear(t *testing.T) {
	assert.Equal(t, 18, fullYears(date(2000, time.February, 29), date(2018, time.February, 28)))
}
