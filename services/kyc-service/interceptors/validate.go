package interceptors

import (
	"context"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

func Validate() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := validate(req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func validate(req any) error {
	switch in := req.(type) {
	case *kycv1.SubmitApplicationRequest:
		return notEmpty(in.UserId, domain.ErrUserIDMustBeUUID)
	case *kycv1.GetApplicationByIDRequest:
		return notEmpty(in.ApplicationId, domain.ErrApplicationIDMustBeUUID)
	case *kycv1.GetApplicationByUserIDRequest:
		return notEmpty(in.UserId, domain.ErrUserIDMustBeUUID)
	case *kycv1.ListApplicationsRequest:
		return validatePage(in.Limit, in.Offset)
	case *kycv1.ClaimApplicationRequest:
		return validateReview(in.ApplicationId, in.OfficerId)
	case *kycv1.ApproveApplicationRequest:
		if err := validateReview(in.ApplicationId, in.OfficerId); err != nil {
			return err
		}

		return validateComment(in.Comment)
	case *kycv1.RejectApplicationRequest:
		if err := validateReview(in.ApplicationId, in.OfficerId); err != nil {
			return err
		}

		if strings.TrimSpace(in.Reason) == "" {
			return domain.ErrReasonIsRequired
		}

		return validateComment(in.Reason)
	case *kycv1.RequestMoreDocumentsRequest:
		if err := validateReview(in.ApplicationId, in.OfficerId); err != nil {
			return err
		}

		if len(in.Types) == 0 {
			return domain.ErrTypesRequired
		}

		for _, docType := range in.Types {
			if docType == kycv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED {
				return domain.ErrTypesRequired
			}
		}

		return validateComment(in.Comment)
	default:
		return nil
	}
}

func notEmpty(value string, err error) error {
	if strings.TrimSpace(value) == "" {
		return err
	}

	return nil
}

func validateReview(applicationID, officerID string) error {
	if err := notEmpty(applicationID, domain.ErrApplicationIDMustBeUUID); err != nil {
		return err
	}

	return notEmpty(officerID, domain.ErrOfficerIDMustBeUUID)
}

func validateComment(comment string) error {
	if utf8.RuneCountInString(comment) > domain.COMMENT_MAX_LENGTH {
		return domain.ErrCommentIsTooLong
	}

	return nil
}

func validatePage(limit, offset int32) error {
	switch {
	case limit < 0:
		return domain.ErrLimitIsNegative
	case offset < 0:
		return domain.ErrOffsetIsNegative
	default:
		return nil
	}
}
