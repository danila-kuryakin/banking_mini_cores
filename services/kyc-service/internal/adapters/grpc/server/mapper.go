package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

var statusPairs = []struct {
	domain models.ApplicationStatus
	proto  kycv1.ApplicationStatus
}{
	{models.STATUS_SUBMITTED, kycv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED},
	{models.STATUS_AUTO_CHECKS, kycv1.ApplicationStatus_APPLICATION_STATUS_AUTO_CHECKS},
	{models.STATUS_AUTO_REJECTED, kycv1.ApplicationStatus_APPLICATION_STATUS_AUTO_REJECTED},
	{models.STATUS_IN_REVIEW, kycv1.ApplicationStatus_APPLICATION_STATUS_IN_REVIEW},
	{models.STATUS_DOCS_REQUESTED, kycv1.ApplicationStatus_APPLICATION_STATUS_DOCS_REQUESTED},
	{models.STATUS_APPROVED, kycv1.ApplicationStatus_APPLICATION_STATUS_APPROVED},
	{models.STATUS_REJECTED, kycv1.ApplicationStatus_APPLICATION_STATUS_REJECTED},
}

var documentTypePairs = []struct {
	domain models.DocumentType
	proto  kycv1.DocumentType
}{
	{models.DOCUMENT_TYPE_PASSPORT, kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT},
	{models.DOCUMENT_TYPE_SELFIE, kycv1.DocumentType_DOCUMENT_TYPE_SELFIE},
	{models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, kycv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS},
}

var checkTypePairs = []struct {
	domain models.CheckType
	proto  kycv1.CheckType
}{
	{models.CHECK_TYPE_SANCTIONS, kycv1.CheckType_CHECK_TYPE_SANCTIONS},
	{models.CHECK_TYPE_AGE, kycv1.CheckType_CHECK_TYPE_AGE},
}

var checkResultPairs = []struct {
	domain models.CheckResult
	proto  kycv1.CheckResult
}{
	{models.CHECK_RESULT_PASSED, kycv1.CheckResult_CHECK_RESULT_PASSED},
	{models.CHECK_RESULT_FAILED, kycv1.CheckResult_CHECK_RESULT_FAILED},
}

var (
	statusToProtoMap, statusFromProtoMap             = buildStatusMaps()
	documentTypeToProtoMap, documentTypeFromProtoMap = buildDocumentTypeMaps()
	checkTypeToProtoMap                              = buildCheckTypeMap()
	checkResultToProtoMap                            = buildCheckResultMap()
)

func buildStatusMaps() (map[models.ApplicationStatus]kycv1.ApplicationStatus, map[kycv1.ApplicationStatus]models.ApplicationStatus) {
	toProto := make(map[models.ApplicationStatus]kycv1.ApplicationStatus, len(statusPairs))
	fromProto := make(map[kycv1.ApplicationStatus]models.ApplicationStatus, len(statusPairs))

	for _, pair := range statusPairs {
		toProto[pair.domain] = pair.proto
		fromProto[pair.proto] = pair.domain
	}

	return toProto, fromProto
}

func buildDocumentTypeMaps() (map[models.DocumentType]kycv1.DocumentType, map[kycv1.DocumentType]models.DocumentType) {
	toProto := make(map[models.DocumentType]kycv1.DocumentType, len(documentTypePairs))
	fromProto := make(map[kycv1.DocumentType]models.DocumentType, len(documentTypePairs))

	for _, pair := range documentTypePairs {
		toProto[pair.domain] = pair.proto
		fromProto[pair.proto] = pair.domain
	}

	return toProto, fromProto
}

func buildCheckTypeMap() map[models.CheckType]kycv1.CheckType {
	out := make(map[models.CheckType]kycv1.CheckType, len(checkTypePairs))

	for _, pair := range checkTypePairs {
		out[pair.domain] = pair.proto
	}

	return out
}

func buildCheckResultMap() map[models.CheckResult]kycv1.CheckResult {
	out := make(map[models.CheckResult]kycv1.CheckResult, len(checkResultPairs))

	for _, pair := range checkResultPairs {
		out[pair.domain] = pair.proto
	}

	return out
}

func applicationToProto(app *models.Application) *kycv1.Application {
	out := &kycv1.Application{
		Id:             app.ID.String(),
		UserId:         app.UserID.String(),
		Status:         statusToProtoMap[app.Status],
		Reason:         app.Reason,
		RequestedTypes: documentTypesToProto(app.RequestedTypes),
		SubmittedAt:    timestamppb.New(app.SubmittedAt),
		CreatedAt:      timestamppb.New(app.CreatedAt),
		UpdatedAt:      timestamppb.New(app.UpdatedAt),
		Checks:         checksToProto(app.Checks),
		Events:         eventsToProto(app.Events),
	}

	if app.Reviewer != nil {
		out.ReviewerId = app.Reviewer.String()
	}

	if app.DecidedAt != nil {
		out.DecidedAt = timestamppb.New(*app.DecidedAt)
	}

	return out
}

func documentTypesToProto(types []models.DocumentType) []kycv1.DocumentType {
	out := make([]kycv1.DocumentType, 0, len(types))

	for _, docType := range types {
		out = append(out, documentTypeToProtoMap[docType])
	}

	return out
}

func documentTypesFromProto(types []kycv1.DocumentType) []models.DocumentType {
	out := make([]models.DocumentType, 0, len(types))

	for _, docType := range types {
		out = append(out, documentTypeFromProtoMap[docType])
	}

	return out
}

func checksToProto(checks []models.Check) []*kycv1.KycCheck {
	out := make([]*kycv1.KycCheck, 0, len(checks))

	for _, check := range checks {
		out = append(out, &kycv1.KycCheck{
			Id:        check.ID.String(),
			Type:      checkTypeToProtoMap[check.Type],
			Result:    checkResultToProtoMap[check.Result],
			Details:   check.Details,
			CreatedAt: timestamppb.New(check.CreatedAt),
		})
	}

	return out
}

func eventsToProto(events []models.Event) []*kycv1.KycEvent {
	out := make([]*kycv1.KycEvent, 0, len(events))

	for _, event := range events {
		item := &kycv1.KycEvent{
			Id:         event.ID.String(),
			FromStatus: statusToProtoMap[event.From],
			ToStatus:   statusToProtoMap[event.To],
			Comment:    event.Comment,
			CreatedAt:  timestamppb.New(event.CreatedAt),
		}

		if event.ActorID != nil {
			item.ActorId = event.ActorID.String()
		}

		out = append(out, item)
	}

	return out
}

func statusesFromProto(statuses []kycv1.ApplicationStatus) []models.ApplicationStatus {
	out := make([]models.ApplicationStatus, 0, len(statuses))

	for _, protoStatus := range statuses {
		if domainStatus, ok := statusFromProtoMap[protoStatus]; ok {
			out = append(out, domainStatus)
		}
	}

	return out
}
