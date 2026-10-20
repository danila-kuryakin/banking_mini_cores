package interceptors

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
)

const (
	TEST_USER_ID  = "11111111-1111-1111-1111-111111111111"
	TEST_ACTOR_ID = "22222222-2222-2222-2222-222222222222"
)

func setStatusRequest(mutate func(in *customerv1.SetStatusRequest)) *customerv1.SetStatusRequest {
	in := &customerv1.SetStatusRequest{
		UserId:  TEST_USER_ID,
		Status:  customerv1.CustomerStatus_ACTIVE,
		ActorId: TEST_ACTOR_ID,
	}
	mutate(in)
	return in
}

func TestValidate_Requests(t *testing.T) {
	tests := []struct {
		name    string
		req     any
		wantErr error
	}{
		{name: "создание, плохой user_id", req: &customerv1.CreateProfileRequest{UserId: "bad"}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "карточка, плохой user_id", req: &customerv1.GetCustomerRequest{UserId: "bad"}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "статус клиента, плохой user_id", req: &customerv1.GetCustomerStatusRequest{UserId: "bad"}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "анкета, плохой user_id", req: &customerv1.UpdateProfileRequest{UserId: "bad"}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "смена статуса, плохой user_id", req: setStatusRequest(func(in *customerv1.SetStatusRequest) { in.UserId = "bad" }), wantErr: domain.ErrUserIDMustBeUUID},

		{name: "создание, валидный", req: &customerv1.CreateProfileRequest{UserId: TEST_USER_ID}},
		{name: "карточка, валидный", req: &customerv1.GetCustomerRequest{UserId: TEST_USER_ID}},
		{name: "статус клиента, валидный", req: &customerv1.GetCustomerStatusRequest{UserId: TEST_USER_ID}},
		{name: "анкета, валидный", req: &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID}},
		{name: "смена статуса, валидный", req: setStatusRequest(func(in *customerv1.SetStatusRequest) {})},

		// Анкету интерцептор не проверяет, это делает сервис.
		{name: "анкета с неверными полями", req: &customerv1.UpdateProfileRequest{
			UserId:  TEST_USER_ID,
			Profile: &customerv1.Profile{Phone: "abc", BirthDate: &date.Date{Year: 3000, Month: 1, Day: 1}},
		}},
		{name: "анкета nil", req: &customerv1.UpdateProfileRequest{UserId: TEST_USER_ID, Profile: nil}},

		{name: "статус не задан", req: setStatusRequest(func(in *customerv1.SetStatusRequest) { in.Status = customerv1.CustomerStatus_UNSPECIFIED }), wantErr: domain.ErrStatusRequired},
		{name: "плохой user_id раньше статуса", req: setStatusRequest(func(in *customerv1.SetStatusRequest) {
			in.UserId = "bad"
			in.Status = customerv1.CustomerStatus_UNSPECIFIED
		}), wantErr: domain.ErrUserIDMustBeUUID},
		{name: "actor_id пустой", req: setStatusRequest(func(in *customerv1.SetStatusRequest) { in.ActorId = "" })},
		{name: "actor_id плохой", req: setStatusRequest(func(in *customerv1.SetStatusRequest) { in.ActorId = "bad" }), wantErr: domain.ErrActorIDMustBeUUID},
		{name: "причина 500 символов", req: setStatusRequest(func(in *customerv1.SetStatusRequest) { in.Reason = strings.Repeat("я", 500) })},
		{name: "причина 501 символ", req: setStatusRequest(func(in *customerv1.SetStatusRequest) { in.Reason = strings.Repeat("я", 501) }), wantErr: domain.ErrReasonIsTooLong},

		{name: "страница по умолчанию", req: &customerv1.ListCustomersRequest{Limit: 0, Offset: 0}},
		{name: "страница задана", req: &customerv1.ListCustomersRequest{Limit: 10, Offset: 5}},
		{name: "отрицательный limit", req: &customerv1.ListCustomersRequest{Limit: -1, Offset: 0}, wantErr: domain.ErrLimitIsNegative},
		{name: "отрицательный offset", req: &customerv1.ListCustomersRequest{Limit: 0, Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "оба отрицательные", req: &customerv1.ListCustomersRequest{Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative},

		{name: "неизвестный тип", req: &customerv1.Customer{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.req)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestValidate_InterceptorRejects(t *testing.T) {
	calls := 0
	handler := func(ctx context.Context, req any) (any, error) {
		calls++
		return "response", nil
	}

	resp, err := Validate()(context.Background(), &customerv1.GetCustomerRequest{UserId: "bad"}, &grpc.UnaryServerInfo{}, handler)

	require.ErrorIs(t, err, domain.ErrUserIDMustBeUUID)
	assert.Nil(t, resp)
	assert.Zero(t, calls, "handler must not be called")
}

func TestValidate_InterceptorPassesThrough(t *testing.T) {
	handlerErr := errors.New("handler failed")
	calls := 0
	handler := func(ctx context.Context, req any) (any, error) {
		calls++
		return "response", handlerErr
	}

	resp, err := Validate()(context.Background(), &customerv1.GetCustomerRequest{UserId: TEST_USER_ID}, &grpc.UnaryServerInfo{}, handler)

	require.ErrorIs(t, err, handlerErr)
	assert.Equal(t, "response", resp)
	assert.Equal(t, 1, calls, "handler must be called once")
}
