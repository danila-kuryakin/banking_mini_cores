package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocumentType_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		docType DocumentType
		want    bool
	}{
		{name: "passport", docType: DOCUMENT_TYPE_PASSPORT, want: true},
		{name: "selfie", docType: DOCUMENT_TYPE_SELFIE, want: true},
		{name: "proof_of_address", docType: DOCUMENT_TYPE_PROOF_OF_ADDRESS, want: true},
		{name: "unspecified", docType: DOCUMENT_TYPE_UNSPECIFIED, want: false},
		{name: "upper_case", docType: "PASSPORT", want: false},
		{name: "unknown", docType: "driver_license", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.docType.IsValid())
		})
	}
}
