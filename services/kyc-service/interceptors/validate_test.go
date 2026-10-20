package interceptors

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

const (
	TEST_USER_ID        = "u"
	TEST_APPLICATION_ID = "a"
	TEST_OFFICER_ID     = "o"
	TEST_UUID           = "6f1c0b0e-3f4a-4b7a-9c3e-2d5f8a1b9e01"
	TEST_NOT_UUID       = "abc"
	TEST_BLANK          = "   "
	TEST_FULL_METHOD    = "/kyc.v1.KycService/SubmitApplication"
	TEST_CTX_VALUE      = "ctx-marker"
)

// testCtxKey - ключ метки контекста, по которой проверяется проброс ctx в хендлер
type testCtxKey struct{}

// validateCase - кейс прямой проверки validate
type validateCase struct {
	name    string
	req     any
	wantErr error
}

// runValidateCases прогоняет кейсы validate: ошибка - sentinel и код InvalidArgument, иначе nil
func runValidateCases(t *testing.T, cases []validateCase) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

// validDate - заведомо корректная дата рождения в прошлом
func validDate() *date.Date {
	return &date.Date{Year: 1980, Month: 1, Day: 15}
}

// dateOf переводит момент времени в date.Date (только год, месяц, день)
func dateOf(moment time.Time) *date.Date {
	return &date.Date{Year: int32(moment.Year()), Month: int32(moment.Month()), Day: int32(moment.Day())}
}

func TestValidate_RejectsInvalidRequest(t *testing.T) {
	h := &fakeHandler{resp: &kycv1.Application{Id: "x"}}
	ctx := context.WithValue(t.Context(), testCtxKey{}, TEST_CTX_VALUE)
	req := &kycv1.SubmitApplicationRequest{UserId: ""}

	resp, err := Validate()(ctx, req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, h.handle)

	require.ErrorIs(t, err, domain.ErrUserIDMustBeUUID)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Nil(t, resp)
	assert.Nil(t, h.calls)
}

func TestValidate_PassesToHandler(t *testing.T) {
	tests := []struct {
		name string
		req  any
	}{
		{name: "valid submit", req: &kycv1.SubmitApplicationRequest{UserId: TEST_UUID}},
		{name: "unknown request type", req: &kycv1.Application{}},
		{name: "nil request", req: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantResp := &kycv1.Application{Id: "x"}
			h := &fakeHandler{resp: wantResp}
			ctx := context.WithValue(t.Context(), testCtxKey{}, TEST_CTX_VALUE)

			resp, err := Validate()(ctx, tt.req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, h.handle)

			require.NoError(t, err)
			assert.Same(t, wantResp, resp)
			assert.Equal(t, []handlerCall{{ctx: ctx, req: tt.req}}, h.calls)
		})
	}
}

func TestValidate_ReturnsHandlerError(t *testing.T) {
	handlerErr := status.Error(codes.NotFound, "boom")
	h := &fakeHandler{err: handlerErr}
	ctx := context.WithValue(t.Context(), testCtxKey{}, TEST_CTX_VALUE)
	req := &kycv1.SubmitApplicationRequest{UserId: TEST_UUID}

	resp, err := Validate()(ctx, req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, h.handle)

	require.ErrorIs(t, err, handlerErr)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Nil(t, resp)
	assert.Equal(t, []handlerCall{{ctx: ctx, req: req}}, h.calls)
}

func TestValidateUnknownRequest(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "application message", req: &kycv1.Application{}},
		{name: "nil", req: nil},
		{name: "string", req: "string"},
	})
}

func TestValidateSubmitApplication(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "empty user id", req: &kycv1.SubmitApplicationRequest{UserId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "blank user id", req: &kycv1.SubmitApplicationRequest{UserId: TEST_BLANK}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "not uuid passes", req: &kycv1.SubmitApplicationRequest{UserId: TEST_NOT_UUID}},
		{name: "uuid", req: &kycv1.SubmitApplicationRequest{UserId: TEST_UUID}},
	})
}

func TestValidateGetApplicationByID(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "empty application id", req: &kycv1.GetApplicationByIDRequest{ApplicationId: ""}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "blank application id", req: &kycv1.GetApplicationByIDRequest{ApplicationId: TEST_BLANK}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "not uuid passes", req: &kycv1.GetApplicationByIDRequest{ApplicationId: TEST_NOT_UUID}},
	})
}

func TestValidateGetApplicationByUserID(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "empty user id", req: &kycv1.GetApplicationByUserIDRequest{UserId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "blank user id", req: &kycv1.GetApplicationByUserIDRequest{UserId: TEST_BLANK}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "not uuid passes", req: &kycv1.GetApplicationByUserIDRequest{UserId: TEST_NOT_UUID}},
	})
}

func TestValidateOfficerActionIDs(t *testing.T) {
	passport := []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT}
	runValidateCases(t, []validateCase{
		{name: "claim empty application id", req: &kycv1.ClaimApplicationRequest{ApplicationId: "", OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "approve empty application id", req: &kycv1.ApproveApplicationRequest{ApplicationId: "", OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "reject empty application id", req: &kycv1.RejectApplicationRequest{ApplicationId: "", OfficerId: TEST_OFFICER_ID, Reason: "r"}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "docs empty application id", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: "", OfficerId: TEST_OFFICER_ID, Types: passport}, wantErr: domain.ErrApplicationIDMustBeUUID},

		{name: "claim blank application id", req: &kycv1.ClaimApplicationRequest{ApplicationId: TEST_BLANK, OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "approve blank application id", req: &kycv1.ApproveApplicationRequest{ApplicationId: TEST_BLANK, OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "reject blank application id", req: &kycv1.RejectApplicationRequest{ApplicationId: TEST_BLANK, OfficerId: TEST_OFFICER_ID, Reason: "r"}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "docs blank application id", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_BLANK, OfficerId: TEST_OFFICER_ID, Types: passport}, wantErr: domain.ErrApplicationIDMustBeUUID},

		{name: "claim empty officer id", req: &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: ""}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "approve empty officer id", req: &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: ""}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "reject empty officer id", req: &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: "", Reason: "r"}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "docs empty officer id", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: "", Types: passport}, wantErr: domain.ErrOfficerIDMustBeUUID},

		{name: "claim both empty", req: &kycv1.ClaimApplicationRequest{}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "approve both empty", req: &kycv1.ApproveApplicationRequest{}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "reject both empty", req: &kycv1.RejectApplicationRequest{Reason: "r"}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "docs both empty", req: &kycv1.RequestMoreDocumentsRequest{Types: passport}, wantErr: domain.ErrApplicationIDMustBeUUID},

		{name: "claim not uuid passes", req: &kycv1.ClaimApplicationRequest{ApplicationId: TEST_NOT_UUID, OfficerId: TEST_NOT_UUID}},
		{name: "approve not uuid passes", req: &kycv1.ApproveApplicationRequest{ApplicationId: TEST_NOT_UUID, OfficerId: TEST_NOT_UUID}},
		{name: "reject not uuid passes", req: &kycv1.RejectApplicationRequest{ApplicationId: TEST_NOT_UUID, OfficerId: TEST_NOT_UUID, Reason: "r"}},
		{name: "docs not uuid passes", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_NOT_UUID, OfficerId: TEST_NOT_UUID, Types: passport}},

		{name: "claim blank officer id", req: &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BLANK}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "approve blank officer id", req: &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BLANK}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "reject blank officer id", req: &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BLANK, Reason: "r"}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "docs blank officer id", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_BLANK, Types: passport}, wantErr: domain.ErrOfficerIDMustBeUUID},
	})
}

func TestValidateListApplications(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "unassigned with reviewer", req: &kycv1.ListApplicationsRequest{Unassigned: true, ReviewerId: "x"}, wantErr: domain.ErrReviewerFilterConflict},
		{name: "conflict before negative limit", req: &kycv1.ListApplicationsRequest{Unassigned: true, ReviewerId: "x", Limit: -1}, wantErr: domain.ErrReviewerFilterConflict},
		{name: "unassigned only", req: &kycv1.ListApplicationsRequest{Unassigned: true}},
		{name: "reviewer only", req: &kycv1.ListApplicationsRequest{ReviewerId: "x"}},
		{name: "negative limit", req: &kycv1.ListApplicationsRequest{Limit: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "negative offset", req: &kycv1.ListApplicationsRequest{Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "both negative", req: &kycv1.ListApplicationsRequest{Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "min int32 limit", req: &kycv1.ListApplicationsRequest{Limit: math.MinInt32}, wantErr: domain.ErrLimitIsNegative},
		{name: "zero page", req: &kycv1.ListApplicationsRequest{}},
		{name: "large page with status", req: &kycv1.ListApplicationsRequest{
			Statuses: []kycv1.ApplicationStatus{kycv1.ApplicationStatus_APPLICATION_STATUS_IN_REVIEW},
			Limit:    1000,
			Offset:   1000,
		}},
	})
}

func TestValidateListSanctions(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "negative limit", req: &kycv1.ListSanctionsRequest{Limit: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "negative offset", req: &kycv1.ListSanctionsRequest{Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "both negative", req: &kycv1.ListSanctionsRequest{Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "min int32 limit", req: &kycv1.ListSanctionsRequest{Limit: math.MinInt32}, wantErr: domain.ErrLimitIsNegative},
		{name: "zero page", req: &kycv1.ListSanctionsRequest{}},
		{name: "large page", req: &kycv1.ListSanctionsRequest{Limit: 1000, Offset: 1000}},
	})
}

func TestValidateApproveApplication(t *testing.T) {
	approve := func(comment string) *kycv1.ApproveApplicationRequest {
		return &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: comment}
	}
	runValidateCases(t, []validateCase{
		{name: "empty comment", req: approve("")},
		{name: "500 ascii runes", req: approve(strings.Repeat("a", domain.COMMENT_MAX_LENGTH))},
		{name: "501 ascii runes", req: approve(strings.Repeat("a", domain.COMMENT_MAX_LENGTH+1)), wantErr: domain.ErrCommentIsTooLong},
		// 500 рун кириллицей = 1000 байт
		{name: "500 cyrillic runes", req: approve(strings.Repeat("я", domain.COMMENT_MAX_LENGTH))},
		{name: "501 cyrillic runes", req: approve(strings.Repeat("я", domain.COMMENT_MAX_LENGTH+1)), wantErr: domain.ErrCommentIsTooLong},
		// 250 кириллических + 1 латинская = 251 руна, 501 байт
		{name: "501 bytes under rune limit", req: approve(strings.Repeat("я", 250) + "a")},
		{name: "ids before comment", req: &kycv1.ApproveApplicationRequest{
			ApplicationId: "",
			OfficerId:     TEST_OFFICER_ID,
			Comment:       strings.Repeat("a", domain.COMMENT_MAX_LENGTH+1),
		}, wantErr: domain.ErrApplicationIDMustBeUUID},
	})
}

func TestValidateRejectApplication(t *testing.T) {
	reject := func(reason string) *kycv1.RejectApplicationRequest {
		return &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Reason: reason}
	}
	runValidateCases(t, []validateCase{
		{name: "empty reason", req: reject(""), wantErr: domain.ErrReasonIsRequired},
		{name: "blank reason", req: reject(TEST_BLANK), wantErr: domain.ErrReasonIsRequired},
		{name: "500 cyrillic runes", req: reject(strings.Repeat("я", domain.COMMENT_MAX_LENGTH))},
		{name: "501 cyrillic runes", req: reject(strings.Repeat("я", domain.COMMENT_MAX_LENGTH+1)), wantErr: domain.ErrCommentIsTooLong},
		{name: "501 ascii runes", req: reject(strings.Repeat("a", domain.COMMENT_MAX_LENGTH+1)), wantErr: domain.ErrCommentIsTooLong},
		{name: "normal reason", req: reject("bad docs")},
		{name: "application id before reason", req: &kycv1.RejectApplicationRequest{ApplicationId: "", OfficerId: TEST_OFFICER_ID, Reason: ""}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "officer id before reason", req: &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: "", Reason: ""}, wantErr: domain.ErrOfficerIDMustBeUUID},
	})
}

func TestValidateRequestMoreDocuments(t *testing.T) {
	docs := func(types []kycv1.DocumentType, comment string) *kycv1.RequestMoreDocumentsRequest {
		return &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Types: types, Comment: comment}
	}
	passport := []kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT}
	runValidateCases(t, []validateCase{
		{name: "nil types", req: docs(nil, ""), wantErr: domain.ErrTypesRequired},
		{name: "empty types", req: docs([]kycv1.DocumentType{}, ""), wantErr: domain.ErrTypesRequired},
		{name: "only unspecified", req: docs([]kycv1.DocumentType{kycv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED}, ""), wantErr: domain.ErrTypesRequired},
		{name: "unspecified among valid", req: docs([]kycv1.DocumentType{
			kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT,
			kycv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED,
			kycv1.DocumentType_DOCUMENT_TYPE_SELFIE,
		}, ""), wantErr: domain.ErrTypesRequired},
		{name: "all valid types", req: docs([]kycv1.DocumentType{
			kycv1.DocumentType_DOCUMENT_TYPE_PASSPORT,
			kycv1.DocumentType_DOCUMENT_TYPE_SELFIE,
			kycv1.DocumentType_DOCUMENT_TYPE_PROOF_OF_ADDRESS,
		}, "")},
		{name: "unknown enum value passes", req: docs([]kycv1.DocumentType{kycv1.DocumentType(99)}, "")},
		{name: "500 cyrillic comment", req: docs(passport, strings.Repeat("я", domain.COMMENT_MAX_LENGTH))},
		{name: "501 cyrillic comment", req: docs(passport, strings.Repeat("я", domain.COMMENT_MAX_LENGTH+1)), wantErr: domain.ErrCommentIsTooLong},
		{name: "ids before types", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: "", OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "types before comment", req: docs(nil, strings.Repeat("a", domain.COMMENT_MAX_LENGTH+1)), wantErr: domain.ErrTypesRequired},
	})
}

func TestValidateAddSanctionEntry(t *testing.T) {
	sanction := func(firstName, lastName, source string, birthDate *date.Date) *kycv1.AddSanctionEntryRequest {
		return &kycv1.AddSanctionEntryRequest{FirstName: firstName, LastName: lastName, Source: source, BirthDate: birthDate}
	}
	longName := strings.Repeat("я", domain.NAME_MAX_LENGTH+1)
	longSource := strings.Repeat("я", domain.SOURCE_MAX_LENGTH+1)
	runValidateCases(t, []validateCase{
		{name: "all empty", req: sanction("", "", "", nil), wantErr: domain.ErrFirstNameIsRequired},
		{name: "blank first name", req: sanction(TEST_BLANK, "P", "", validDate()), wantErr: domain.ErrFirstNameIsRequired},
		{name: "empty last name", req: sanction("Ivan", "", "", validDate()), wantErr: domain.ErrLastNameIsRequired},
		{name: "blank last name", req: sanction("Ivan", TEST_BLANK, "", validDate()), wantErr: domain.ErrLastNameIsRequired},
		{name: "first name 101 runes", req: sanction(longName, "P", "", validDate()), wantErr: domain.ErrNameIsTooLong},
		{name: "first name 100 runes", req: sanction(strings.Repeat("я", domain.NAME_MAX_LENGTH), "P", "", validDate())},
		{name: "last name 101 runes", req: sanction("Ivan", strings.Repeat("a", domain.NAME_MAX_LENGTH+1), "", validDate()), wantErr: domain.ErrNameIsTooLong},
		{name: "last name 100 runes", req: sanction("Ivan", strings.Repeat("я", domain.NAME_MAX_LENGTH), "", validDate())},
		{name: "required before length", req: sanction(strings.Repeat("a", domain.NAME_MAX_LENGTH+1), "", "", validDate()), wantErr: domain.ErrLastNameIsRequired},
		{name: "source 200 runes", req: sanction("Ivan", "Petrov", strings.Repeat("я", domain.SOURCE_MAX_LENGTH), validDate())},
		{name: "source 201 runes", req: sanction("Ivan", "Petrov", longSource, validDate()), wantErr: domain.ErrSourceIsTooLong},
		{name: "name length before source", req: sanction(longName, "Petrov", longSource, validDate()), wantErr: domain.ErrNameIsTooLong},
		{name: "source before birth date", req: sanction("Ivan", "Petrov", longSource, nil), wantErr: domain.ErrSourceIsTooLong},
		{name: "first name before birth date", req: sanction("", "Petrov", "", &date.Date{Year: 2001, Month: 2, Day: 29}), wantErr: domain.ErrFirstNameIsRequired},
		{name: "valid entry", req: sanction("Ivan", "Petrov", "", validDate())},
	})
}

func TestValidateSanctionBirthDate(t *testing.T) {
	now := time.Now().UTC()
	withDate := func(birthDate *date.Date) *kycv1.AddSanctionEntryRequest {
		return &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "Petrov", BirthDate: birthDate}
	}
	runValidateCases(t, []validateCase{
		{name: "nil date", req: withDate(nil), wantErr: domain.ErrBirthDateIsRequired},
		{name: "zero date", req: withDate(&date.Date{}), wantErr: domain.ErrBirthDateIsRequired},
		{name: "zero year", req: withDate(&date.Date{Year: 0, Month: 1, Day: 15}), wantErr: domain.ErrBirthDateIsRequired},
		{name: "zero month", req: withDate(&date.Date{Year: 1980, Month: 0, Day: 15}), wantErr: domain.ErrBirthDateIsRequired},
		{name: "zero day", req: withDate(&date.Date{Year: 1980, Month: 1, Day: 0}), wantErr: domain.ErrBirthDateIsRequired},
		{name: "feb 29 non leap", req: withDate(&date.Date{Year: 2001, Month: 2, Day: 29}), wantErr: domain.ErrBirthDateIsNotValid},
		{name: "april 31", req: withDate(&date.Date{Year: 2023, Month: 4, Day: 31}), wantErr: domain.ErrBirthDateIsNotValid},
		{name: "month 13", req: withDate(&date.Date{Year: 1980, Month: 13, Day: 1}), wantErr: domain.ErrBirthDateIsNotValid},
		{name: "day 32", req: withDate(&date.Date{Year: 1980, Month: 1, Day: 32}), wantErr: domain.ErrBirthDateIsNotValid},
		// 1900 делится на 100, но не на 400 - не високосный
		{name: "feb 29 1900", req: withDate(&date.Date{Year: 1900, Month: 2, Day: 29}), wantErr: domain.ErrBirthDateIsNotValid},
		{name: "feb 29 leap", req: withDate(&date.Date{Year: 2000, Month: 2, Day: 29})},
		{name: "invalid before future", req: withDate(&date.Date{Year: 2999, Month: 2, Day: 30}), wantErr: domain.ErrBirthDateIsNotValid},
		{name: "tomorrow utc", req: withDate(dateOf(now.AddDate(0, 0, 1))), wantErr: domain.ErrBirthDateInFuture},
		{name: "today utc", req: withDate(dateOf(now))},
	})
}

func TestValidateErrorsAreInvalidArgument(t *testing.T) {
	runValidateCases(t, []validateCase{
		{name: "user id", req: &kycv1.SubmitApplicationRequest{}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "application id", req: &kycv1.GetApplicationByIDRequest{}, wantErr: domain.ErrApplicationIDMustBeUUID},
		{name: "officer id", req: &kycv1.ClaimApplicationRequest{ApplicationId: TEST_APPLICATION_ID}, wantErr: domain.ErrOfficerIDMustBeUUID},
		{name: "reviewer conflict", req: &kycv1.ListApplicationsRequest{Unassigned: true, ReviewerId: "x"}, wantErr: domain.ErrReviewerFilterConflict},
		{name: "limit", req: &kycv1.ListSanctionsRequest{Limit: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "offset", req: &kycv1.ListSanctionsRequest{Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "comment", req: &kycv1.ApproveApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID, Comment: strings.Repeat("a", domain.COMMENT_MAX_LENGTH+1)}, wantErr: domain.ErrCommentIsTooLong},
		{name: "reason", req: &kycv1.RejectApplicationRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrReasonIsRequired},
		{name: "types", req: &kycv1.RequestMoreDocumentsRequest{ApplicationId: TEST_APPLICATION_ID, OfficerId: TEST_OFFICER_ID}, wantErr: domain.ErrTypesRequired},
		{name: "first name", req: &kycv1.AddSanctionEntryRequest{}, wantErr: domain.ErrFirstNameIsRequired},
		{name: "last name", req: &kycv1.AddSanctionEntryRequest{FirstName: "Ivan"}, wantErr: domain.ErrLastNameIsRequired},
		{name: "name length", req: &kycv1.AddSanctionEntryRequest{FirstName: strings.Repeat("a", domain.NAME_MAX_LENGTH+1), LastName: "P", BirthDate: validDate()}, wantErr: domain.ErrNameIsTooLong},
		{name: "source length", req: &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "P", Source: strings.Repeat("a", domain.SOURCE_MAX_LENGTH+1), BirthDate: validDate()}, wantErr: domain.ErrSourceIsTooLong},
		{name: "birth date required", req: &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "P"}, wantErr: domain.ErrBirthDateIsRequired},
		{name: "birth date not valid", req: &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "P", BirthDate: &date.Date{Year: 2001, Month: 2, Day: 29}}, wantErr: domain.ErrBirthDateIsNotValid},
		{name: "birth date in future", req: &kycv1.AddSanctionEntryRequest{FirstName: "Ivan", LastName: "P", BirthDate: dateOf(time.Now().UTC().AddDate(0, 0, 1))}, wantErr: domain.ErrBirthDateInFuture},
	})
}
