package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck_Passed(t *testing.T) {
	tests := []struct {
		name   string
		result CheckResult
		want   bool
	}{
		{name: "passed", result: CHECK_RESULT_PASSED, want: true},
		{name: "failed", result: CHECK_RESULT_FAILED, want: false},
		{name: "unspecified", result: CHECK_RESULT_UNSPECIFIED, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := Check{Type: CHECK_TYPE_AGE, Result: tt.result}
			assert.Equal(t, tt.want, check.Passed())
		})
	}
}

func TestAllCheckTypes(t *testing.T) {
	assert.Equal(t, []CheckType{CHECK_TYPE_SANCTIONS, CHECK_TYPE_AGE}, AllCheckTypes())
}

func TestAllCheckTypes_ReturnsCopy(t *testing.T) {
	first := AllCheckTypes()
	for i := range first {
		first[i] = "x"
	}
	assert.Equal(t, []CheckType{CHECK_TYPE_SANCTIONS, CHECK_TYPE_AGE}, AllCheckTypes())
}
