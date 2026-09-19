package client

import (
	"time"

	"google.golang.org/genproto/googleapis/type/date"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/customer/v1"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/filestore/v1"
)

func dateFromProto(in *date.Date) *time.Time {
	if in == nil || in.Year == 0 || in.Month == 0 || in.Day == 0 {
		return nil
	}

	parsed := time.Date(int(in.Year), time.Month(in.Month), int(in.Day), 0, 0, 0, 0, time.UTC)

	return &parsed
}

var customerStatusPairs = []struct {
	domain models.CustomerStatus
	proto  customerv1.CustomerStatus
}{
	{models.CUSTOMER_STATUS_NEW, customerv1.CustomerStatus_NEW},
	{models.CUSTOMER_STATUS_PROFILE_FILLED, customerv1.CustomerStatus_PROFILE_FILLED},
	{models.CUSTOMER_STATUS_ON_KYC, customerv1.CustomerStatus_ON_KYC},
	{models.CUSTOMER_STATUS_ACTIVE, customerv1.CustomerStatus_ACTIVE},
	{models.CUSTOMER_STATUS_REJECTED, customerv1.CustomerStatus_REJECTED},
	{models.CUSTOMER_STATUS_BLOCKED, customerv1.CustomerStatus_BLOCKED},
}

var customerStatusToProtoMap, customerStatusFromProtoMap = buildCustomerStatusMaps()

func buildCustomerStatusMaps() (map[models.CustomerStatus]customerv1.CustomerStatus, map[customerv1.CustomerStatus]models.CustomerStatus) {
	toProto := make(map[models.CustomerStatus]customerv1.CustomerStatus, len(customerStatusPairs))
	fromProto := make(map[customerv1.CustomerStatus]models.CustomerStatus, len(customerStatusPairs))

	for _, pair := range customerStatusPairs {
		toProto[pair.domain] = pair.proto
		fromProto[pair.proto] = pair.domain
	}

	return toProto, fromProto
}

func customerStatusToProto(s models.CustomerStatus) customerv1.CustomerStatus {
	return customerStatusToProtoMap[s]
}

func customerStatusFromProto(s customerv1.CustomerStatus) models.CustomerStatus {
	return customerStatusFromProtoMap[s]
}

func documentTypeFromProto(t filestorev1.FileType) models.DocumentType {
	return documentTypeFromProtoMap[t]
}

var documentTypePairs = []struct {
	domain models.DocumentType
	proto  filestorev1.FileType
}{
	{models.DOCUMENT_TYPE_PASSPORT, filestorev1.FileType_FILE_TYPE_PASSPORT},
	{models.DOCUMENT_TYPE_SELFIE, filestorev1.FileType_FILE_TYPE_SELFIE},
	{models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS},
}

var documentTypeFromProtoMap = buildDocumentTypeFromProtoMap()

func buildDocumentTypeFromProtoMap() map[filestorev1.FileType]models.DocumentType {
	fromProto := make(map[filestorev1.FileType]models.DocumentType, len(documentTypePairs))

	for _, pair := range documentTypePairs {
		fromProto[pair.proto] = pair.domain
	}

	return fromProto
}
