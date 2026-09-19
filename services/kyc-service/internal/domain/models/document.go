package models

import "slices"

// DocumentType - тип документа. Совпадает с меткой enum document_type в
// Postgres. В filestore-service свой FileType; перевод живёт в его клиенте.
type DocumentType string

const (
	DOCUMENT_TYPE_UNSPECIFIED      DocumentType = ""
	DOCUMENT_TYPE_PASSPORT         DocumentType = "passport"
	DOCUMENT_TYPE_SELFIE           DocumentType = "selfie"
	DOCUMENT_TYPE_PROOF_OF_ADDRESS DocumentType = "proof_of_address"
)

var allDocumentTypes = []DocumentType{
	DOCUMENT_TYPE_PASSPORT,
	DOCUMENT_TYPE_SELFIE,
	DOCUMENT_TYPE_PROOF_OF_ADDRESS,
}

func (d DocumentType) IsValid() bool {
	return slices.Contains(allDocumentTypes, d)
}
