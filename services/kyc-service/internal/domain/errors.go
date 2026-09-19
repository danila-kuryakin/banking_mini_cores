package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrApplicationNotFound = errors.New("application not found")
	// ErrActiveApplicationExists - у клиента уже есть незакрытая заявка.
	// Ловится частичным уникальным индексом applications_active_user_id_key.
	ErrActiveApplicationExists = errors.New("active application already exists")
	// ErrStatusConflict - статус заявки изменился между проверкой перехода и
	// записью. Ловится условием на исходный статус в UPDATE.
	ErrStatusConflict = errors.New("application status changed concurrently")
	// ErrAlreadyClaimed - заявку уже взял другой офицер.
	ErrAlreadyClaimed = errors.New("application is already claimed")
)

// Ошибки, приходящие от соседних сервисов и из предусловий подачи.
var (
	ErrCustomerNotFound = errors.New("customer not found")
	// ErrProfileNotFilled - анкета не заполнена, подавать заявку не с чем.
	ErrProfileNotFilled = errors.New("customer profile is not filled")
	// ErrDocumentsMissing - не хватает подтверждённых документов.
	ErrDocumentsMissing = errors.New("required documents are missing")
	// ErrReasonRequired - отказ без причины не принимается.
	ErrReasonRequired = errors.New("reason is required")
	// ErrDocumentTypesRequired - запрос документов без списка типов.
	ErrDocumentTypesRequired = errors.New("document types are required")
)

// Ошибки, уходящие наружу gRPC-статусами.
var (
	ErrApplicationIDMustBeUUID = status.Error(codes.InvalidArgument, "application_id must be a uuid")
	ErrUserIDMustBeUUID        = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrOfficerIDMustBeUUID     = status.Error(codes.InvalidArgument, "officer_id must be a uuid")
	ErrReasonIsRequired        = status.Error(codes.InvalidArgument, "reason is required to reject an application")
	ErrTypesRequired           = status.Error(codes.InvalidArgument, "at least one known document type is required")
	ErrCommentIsTooLong        = status.Error(codes.InvalidArgument, "comment is longer than 500 characters")
	ErrLimitIsNegative         = status.Error(codes.InvalidArgument, "limit must not be negative")
	ErrOffsetIsNegative        = status.Error(codes.InvalidArgument, "offset must not be negative")

	ErrApplicationMissing  = status.Error(codes.NotFound, "application not found")
	ErrCustomerMissing     = status.Error(codes.NotFound, "customer not found")
	ErrApplicationExists   = status.Error(codes.AlreadyExists, "customer already has an active application")
	ErrProfileIsNotFilled  = status.Error(codes.FailedPrecondition, "customer profile must be filled before applying")
	ErrDocumentsIncomplete = status.Error(codes.FailedPrecondition, "required documents are not confirmed")
	ErrTransitionForbidden = status.Error(codes.FailedPrecondition, "application cannot move to the requested status")
	ErrClaimedByOther      = status.Error(codes.FailedPrecondition, "application is already claimed by another officer")

	ErrSubmitFailed = status.Error(codes.Internal, "failed to submit application")
	ErrGetFailed    = status.Error(codes.Internal, "failed to get application")
	ErrListFailed   = status.Error(codes.Internal, "failed to list applications")
	ErrReviewFailed = status.Error(codes.Internal, "failed to update application")
)
