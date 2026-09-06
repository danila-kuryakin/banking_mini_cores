package interceptors

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

var phonePattern = regexp.MustCompile(domain.PHONE_PATTERN)

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
	case *customerv1.CreateProfileRequest:
		return validateUserID(in.UserId)
	case *customerv1.GetCustomerRequest:
		return validateUserID(in.UserId)
	case *customerv1.UpdateProfileRequest:
		if err := validateUserID(in.UserId); err != nil {
			return err
		}

		return validateProfile(in.Profile)
	case *customerv1.GetCustomerStatusRequest:
		return validateUserID(in.UserId)
	case *customerv1.ListCustomersRequest:
		return validatePage(in.Limit, in.Offset)
	default:
		return nil
	}
}

func validateUserID(userID string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrUserIDMustBeUUID
	}

	return nil
}

func validateProfile(profile *customerv1.Profile) error {
	if profile == nil {
		return nil
	}

	switch {
	case len(profile.FirstName) > domain.NAME_MAX_LENGTH:
		return domain.ErrFirstNameIsTooLong
	case len(profile.LastName) > domain.NAME_MAX_LENGTH:
		return domain.ErrLastNameIsTooLong
	case len(profile.Citizenship) > domain.CITIZENSHIP_MAX_LENGTH:
		return domain.ErrCitizenshipIsTooLong
	case profile.Phone != "" && !phonePattern.MatchString(profile.Phone):
		return domain.ErrPhoneIsNotValid
	}

	return validateBirthDate(profile.BirthDate)
}

func validateBirthDate(birthDate *date.Date) error {
	if birthDate == nil || (birthDate.Year == 0 && birthDate.Month == 0 && birthDate.Day == 0) {
		return nil
	}

	if birthDate.Year == 0 || birthDate.Month == 0 || birthDate.Day == 0 {
		return domain.ErrBirthDateIsNotValid
	}

	parsed := time.Date(int(birthDate.Year), time.Month(birthDate.Month), int(birthDate.Day), 0, 0, 0, 0, time.UTC)

	if int32(parsed.Year()) != birthDate.Year || int32(parsed.Month()) != birthDate.Month || int32(parsed.Day()) != birthDate.Day {
		return domain.ErrBirthDateIsNotValid
	}

	now := time.Now().UTC()

	switch {
	case parsed.After(now):
		return domain.ErrBirthDateInFuture
	case parsed.Before(now.AddDate(-domain.MAX_CUSTOMER_AGE_YEARS, 0, 0)):
		return domain.ErrBirthDateIsTooOld
	default:
		return nil
	}
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
