package interceptors

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

const (
	TEST_USER_ID     = "3f2b8c4e-1a2d-4b5c-9e8f-0a1b2c3d4e5f"
	TEST_FULL_METHOD = "/notification.v1.NotificationService/ListNotifications"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		req      any
		wantErr  error
		wantCode codes.Code
	}{
		{name: "empty user_id", req: &notificationv1.ListNotificationsRequest{UserId: ""}, wantErr: domain.ErrUserIDMustBeUUID, wantCode: codes.InvalidArgument},
		{name: "spaces user_id", req: &notificationv1.ListNotificationsRequest{UserId: "   "}, wantErr: domain.ErrUserIDMustBeUUID, wantCode: codes.InvalidArgument},
		{name: "whitespace user_id", req: &notificationv1.ListNotificationsRequest{UserId: "\t \n"}, wantErr: domain.ErrUserIDMustBeUUID, wantCode: codes.InvalidArgument},
		{name: "negative limit", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: -1}, wantErr: domain.ErrLimitIsNegative, wantCode: codes.InvalidArgument},
		{name: "min int32 limit", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: math.MinInt32}, wantErr: domain.ErrLimitIsNegative, wantCode: codes.InvalidArgument},
		{name: "negative offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: 0, Offset: -1}, wantErr: domain.ErrOffsetIsNegative, wantCode: codes.InvalidArgument},
		{name: "min int32 offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Offset: math.MinInt32}, wantErr: domain.ErrOffsetIsNegative, wantCode: codes.InvalidArgument},
		{name: "user_id checked before limit and offset", req: &notificationv1.ListNotificationsRequest{UserId: "", Limit: -1, Offset: -1}, wantErr: domain.ErrUserIDMustBeUUID, wantCode: codes.InvalidArgument},
		{name: "user_id checked before offset", req: &notificationv1.ListNotificationsRequest{UserId: "", Limit: 0, Offset: -1}, wantErr: domain.ErrUserIDMustBeUUID, wantCode: codes.InvalidArgument},
		{name: "limit checked before offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative, wantCode: codes.InvalidArgument},
		{name: "zero limit and offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: 0, Offset: 0}},
		{name: "max int32 limit and offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: math.MaxInt32, Offset: math.MaxInt32}},
		{name: "uuid user_id", req: &notificationv1.ListNotificationsRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 40}},
		{name: "non uuid user_id passed", req: &notificationv1.ListNotificationsRequest{UserId: "not-a-uuid"}},
		{name: "user_id with surrounding spaces passed", req: &notificationv1.ListNotificationsRequest{UserId: " u "}},
		{name: "empty message not checked", req: &emptypb.Empty{}},
		{name: "other message not checked", req: &notificationv1.ListNotificationsResponse{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			wantResp := &notificationv1.ListNotificationsResponse{}
			handler := &fakeHandler{resp: wantResp}
			info := &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}

			resp, err := Validate()(ctx, tt.req, info, handler.handle)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, tt.wantCode, status.Code(err))
				assert.Nil(t, resp)
				assert.Equal(t, 0, handler.calls)
				return
			}
			require.NoError(t, err)
			assert.Same(t, wantResp, resp)
			assert.Equal(t, 1, handler.calls)
			assert.Same(t, tt.req, handler.gotReq)
			assert.Equal(t, ctx, handler.gotCtx)
		})
	}
}

// Ошибка хендлера пробрасывается как есть, хендлер вызван один раз с тем же req и ctx
func TestValidate_HandlerErrorPassedThrough(t *testing.T) {
	ctx := t.Context()
	handlerErr := status.Error(codes.Internal, "handler failed")
	handler := &fakeHandler{err: handlerErr}
	req := &notificationv1.ListNotificationsRequest{UserId: TEST_USER_ID}

	resp, err := Validate()(ctx, req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, handler.handle)

	require.ErrorIs(t, err, handlerErr)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Nil(t, resp)
	assert.Equal(t, 1, handler.calls)
	assert.Same(t, req, handler.gotReq)
	assert.Equal(t, ctx, handler.gotCtx)
}

// Инвариант: при ошибке интерцептора хендлер не вызван, ответ nil, код InvalidArgument
func TestValidate_InvariantRejectedRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *notificationv1.ListNotificationsRequest
	}{
		{name: "empty user_id", req: &notificationv1.ListNotificationsRequest{UserId: ""}},
		{name: "spaces user_id", req: &notificationv1.ListNotificationsRequest{UserId: "   "}},
		{name: "whitespace user_id", req: &notificationv1.ListNotificationsRequest{UserId: "\t \n"}},
		{name: "negative limit", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: -1}},
		{name: "min int32 limit", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: math.MinInt32}},
		{name: "negative offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Offset: -1}},
		{name: "min int32 offset", req: &notificationv1.ListNotificationsRequest{UserId: "u", Offset: math.MinInt32}},
		{name: "all invalid", req: &notificationv1.ListNotificationsRequest{UserId: "", Limit: -1, Offset: -1}},
		{name: "user_id and offset invalid", req: &notificationv1.ListNotificationsRequest{UserId: "", Offset: -1}},
		{name: "limit and offset invalid", req: &notificationv1.ListNotificationsRequest{UserId: "u", Limit: -1, Offset: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &fakeHandler{resp: &notificationv1.ListNotificationsResponse{}}

			resp, err := Validate()(t.Context(), tt.req, &grpc.UnaryServerInfo{FullMethod: TEST_FULL_METHOD}, handler.handle)

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, resp)
			assert.Equal(t, 0, handler.calls)
		})
	}
}
