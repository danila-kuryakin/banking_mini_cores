package interceptors

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

const (
	TEST_USER_ID         = "11111111-1111-1111-1111-111111111111"
	TEST_FILE_ID         = "22222222-2222-2222-2222-222222222222"
	TEST_BAD_UUID_ABC    = "abc"
	TEST_BAD_UUID_SHORT  = "11111111-1111-1111-1111-11111111111"
	TEST_BAD_UUID_NONHEX = "11111111-1111-1111-1111-11111111111g"
	TEST_FILENAME        = "passport.pdf"
	TEST_CTX_MARKER      = "validate-test"

	TEST_METHOD_INIT_UPLOAD    = "/filestore.v1.FileService/InitUpload"
	TEST_METHOD_CONFIRM_UPLOAD = "/filestore.v1.FileService/ConfirmUpload"
	TEST_METHOD_GET_DOWNLOAD   = "/filestore.v1.FileService/GetDownloadUrl"
	TEST_METHOD_LIST_FILES     = "/filestore.v1.FileService/ListFiles"
	TEST_METHOD_HAS_REQUIRED   = "/filestore.v1.FileService/HasRequiredFiles"
	TEST_METHOD_UNKNOWN        = "/filestore.v1.FileService/Unknown"
)

// ctxKey - ключ маркера, по которому видно, что до handler дошёл наш ctx
type ctxKey struct{}

// handlerCall - один вызов handler с его аргументами
type handlerCall struct {
	ctx context.Context
	req any
}

// fakeHandler - фейк grpc.UnaryHandler: пишет вызовы и отдаёт заданную пару (resp, err)
type fakeHandler struct {
	resp  any
	err   error
	calls []handlerCall
}

var _ grpc.UnaryHandler = (*fakeHandler)(nil).handle

func (h *fakeHandler) handle(ctx context.Context, req any) (any, error) {
	h.calls = append(h.calls, handlerCall{ctx: ctx, req: req})
	return h.resp, h.err
}

// newTestCtx - контекст теста с маркером
func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	return context.WithValue(t.Context(), ctxKey{}, TEST_CTX_MARKER)
}

// runValidate прогоняет запрос через интерцептор и проверяет исход: вызов handler или ошибку валидации
func runValidate(t *testing.T, method string, req any, wantErr error) {
	t.Helper()
	ctx := newTestCtx(t)
	h := &fakeHandler{resp: &emptypb.Empty{}}

	resp, err := Validate()(ctx, req, &grpc.UnaryServerInfo{FullMethod: method}, h.handle)

	if wantErr != nil {
		require.ErrorIs(t, err, wantErr)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, resp)
		assert.Empty(t, h.calls)
		return
	}
	require.NoError(t, err)
	assert.Same(t, h.resp, resp)
	require.Len(t, h.calls, 1)
	assert.Same(t, ctx, h.calls[0].ctx)
	assert.Same(t, req, h.calls[0].req)
}

func TestValidate_InitUpload(t *testing.T) {
	tests := []struct {
		name    string
		req     *filestorev1.InitUploadRequest
		wantErr error
	}{
		{name: "valid passport", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}},
		{name: "empty filename allowed", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS, Filename: ""}},
		{name: "empty user_id", req: &filestorev1.InitUploadRequest{UserId: "", Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id abc", req: &filestorev1.InitUploadRequest{UserId: TEST_BAD_UUID_ABC, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id 35 chars", req: &filestorev1.InitUploadRequest{UserId: TEST_BAD_UUID_SHORT, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id non-hex", req: &filestorev1.InitUploadRequest{UserId: TEST_BAD_UUID_NONHEX, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "all fields bad - user_id first", req: &filestorev1.InitUploadRequest{UserId: TEST_BAD_UUID_ABC, Type: filestorev1.FileType_FILE_TYPE_UNSPECIFIED, Filename: strings.Repeat("a", 256)}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "type unspecified", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_UNSPECIFIED, Filename: TEST_FILENAME}, wantErr: domain.ErrFileTypeRequired},
		{name: "type unspecified before filename", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_UNSPECIFIED, Filename: strings.Repeat("a", 256)}, wantErr: domain.ErrFileTypeRequired},
		{name: "type 4 unknown", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType(4), Filename: TEST_FILENAME}, wantErr: domain.ErrFileTypeUnknown},
		{name: "type 99 unknown before filename", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType(99), Filename: strings.Repeat("a", 256)}, wantErr: domain.ErrFileTypeUnknown},
		{name: "type -1 unknown", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType(-1), Filename: TEST_FILENAME}, wantErr: domain.ErrFileTypeUnknown},
		{name: "filename 255 ascii", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: strings.Repeat("a", 255)}},
		{name: "filename 256 ascii", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: strings.Repeat("a", 256)}, wantErr: domain.ErrFilenameIsTooLong},
		// 255 рун кириллицы - 510 байт, должно пройти
		{name: "filename 255 cyrillic", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: strings.Repeat("я", 255)}},
		{name: "filename 256 cyrillic", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: strings.Repeat("я", 256)}, wantErr: domain.ErrFilenameIsTooLong},
		// 255 эмодзи по 4 байта - 1020 байт, должно пройти
		{name: "filename 255 emoji", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: strings.Repeat("😀", 255)}},
		{name: "filename 256 emoji", req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: strings.Repeat("😀", 256)}, wantErr: domain.ErrFilenameIsTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidate(t, TEST_METHOD_INIT_UPLOAD, tt.req, tt.wantErr)
		})
	}
}

func TestValidate_ConfirmUpload(t *testing.T) {
	tests := []struct {
		name    string
		req     *filestorev1.ConfirmUploadRequest
		wantErr error
	}{
		{name: "valid", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_FILE_ID}},
		{name: "user_id abc", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_BAD_UUID_ABC, FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty user_id", req: &filestorev1.ConfirmUploadRequest{UserId: "", FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty file_id", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: ""}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id abc", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_UUID_ABC}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id 35 chars", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_UUID_SHORT}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id non-hex", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_UUID_NONHEX}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "both empty - user_id first", req: &filestorev1.ConfirmUploadRequest{UserId: "", FileId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidate(t, TEST_METHOD_CONFIRM_UPLOAD, tt.req, tt.wantErr)
		})
	}
}

func TestValidate_GetDownloadUrl(t *testing.T) {
	tests := []struct {
		name    string
		req     *filestorev1.GetDownloadUrlRequest
		wantErr error
	}{
		{name: "valid", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_FILE_ID}},
		{name: "user_id abc", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_BAD_UUID_ABC, FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty user_id", req: &filestorev1.GetDownloadUrlRequest{UserId: "", FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty file_id", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: ""}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id abc", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_UUID_ABC}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id 35 chars", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_UUID_SHORT}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id non-hex", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_UUID_NONHEX}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "both empty - user_id first", req: &filestorev1.GetDownloadUrlRequest{UserId: "", FileId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidate(t, TEST_METHOD_GET_DOWNLOAD, tt.req, tt.wantErr)
		})
	}
}

func TestValidate_ListFiles(t *testing.T) {
	tests := []struct {
		name    string
		req     *filestorev1.ListFilesRequest
		wantErr error
	}{
		{name: "zero limit and offset", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 0, Offset: 0}},
		{name: "positive limit and offset", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 40}},
		{name: "max int32", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: math.MaxInt32, Offset: math.MaxInt32}},
		{name: "limit -1", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: -1, Offset: 0}, wantErr: domain.ErrLimitIsNegative},
		{name: "limit min int32", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: math.MinInt32, Offset: 0}, wantErr: domain.ErrLimitIsNegative},
		{name: "offset -1", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 0, Offset: -1}, wantErr: domain.ErrOffsetIsNegative},
		{name: "offset min int32", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 0, Offset: math.MinInt32}, wantErr: domain.ErrOffsetIsNegative},
		{name: "both negative - limit first", req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: -1, Offset: -1}, wantErr: domain.ErrLimitIsNegative},
		{name: "all bad - user_id first", req: &filestorev1.ListFilesRequest{UserId: TEST_BAD_UUID_ABC, Limit: -1, Offset: -1}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "empty user_id", req: &filestorev1.ListFilesRequest{UserId: "", Limit: 0, Offset: 0}, wantErr: domain.ErrUserIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidate(t, TEST_METHOD_LIST_FILES, tt.req, tt.wantErr)
		})
	}
}

func TestValidate_HasRequiredFiles(t *testing.T) {
	tests := []struct {
		name    string
		req     *filestorev1.HasRequiredFilesRequest
		wantErr error
	}{
		{name: "valid", req: &filestorev1.HasRequiredFilesRequest{UserId: TEST_USER_ID}},
		{name: "empty user_id", req: &filestorev1.HasRequiredFilesRequest{UserId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id abc", req: &filestorev1.HasRequiredFilesRequest{UserId: TEST_BAD_UUID_ABC}, wantErr: domain.ErrUserIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidate(t, TEST_METHOD_HAS_REQUIRED, tt.req, tt.wantErr)
		})
	}
}

func TestValidate_UnknownRequestPassesThrough(t *testing.T) {
	runValidate(t, TEST_METHOD_UNKNOWN, &emptypb.Empty{}, nil)
}

func TestValidate_HandlerResultReturnedAsIs(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		req         any
		handlerResp any
		handlerErr  error
	}{
		{
			name:       "init upload handler error",
			method:     TEST_METHOD_INIT_UPLOAD,
			req:        &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME},
			handlerErr: status.Error(codes.Internal, "boom"),
		},
		{
			name:       "unknown request handler error",
			method:     TEST_METHOD_UNKNOWN,
			req:        &emptypb.Empty{},
			handlerErr: status.Error(codes.Internal, "boom"),
		},
		{
			name:        "list files handler resp and error together",
			method:      TEST_METHOD_LIST_FILES,
			req:         &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 10, Offset: 0},
			handlerResp: &filestorev1.ListFilesResponse{},
			handlerErr:  status.Error(codes.Internal, "boom"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestCtx(t)
			h := &fakeHandler{resp: tt.handlerResp, err: tt.handlerErr}

			resp, err := Validate()(ctx, tt.req, &grpc.UnaryServerInfo{FullMethod: tt.method}, h.handle)

			require.ErrorIs(t, err, tt.handlerErr)
			assert.Equal(t, codes.Internal, status.Code(err))
			// сравнение интерфейсов: тот же указатель либо оба nil
			assert.True(t, resp == tt.handlerResp, "resp must be exactly the handler's resp")
			require.Len(t, h.calls, 1)
			assert.Same(t, ctx, h.calls[0].ctx)
			assert.Same(t, tt.req, h.calls[0].req)
		})
	}
}

func TestValidate_Invariant_CalledOnceOrInvalidArgument(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		req        any
		wantCalled bool
	}{
		{name: "init upload valid", method: TEST_METHOD_INIT_UPLOAD, req: &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}, wantCalled: true},
		{name: "init upload invalid", method: TEST_METHOD_INIT_UPLOAD, req: &filestorev1.InitUploadRequest{UserId: TEST_BAD_UUID_ABC, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: TEST_FILENAME}},
		{name: "confirm upload valid", method: TEST_METHOD_CONFIRM_UPLOAD, req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_FILE_ID}, wantCalled: true},
		{name: "confirm upload invalid", method: TEST_METHOD_CONFIRM_UPLOAD, req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: ""}},
		{name: "get download url valid", method: TEST_METHOD_GET_DOWNLOAD, req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_FILE_ID}, wantCalled: true},
		{name: "get download url invalid", method: TEST_METHOD_GET_DOWNLOAD, req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: ""}},
		{name: "list files valid", method: TEST_METHOD_LIST_FILES, req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 0, Offset: 0}, wantCalled: true},
		{name: "list files invalid", method: TEST_METHOD_LIST_FILES, req: &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: -1, Offset: 0}},
		{name: "has required files valid", method: TEST_METHOD_HAS_REQUIRED, req: &filestorev1.HasRequiredFilesRequest{UserId: TEST_USER_ID}, wantCalled: true},
		{name: "has required files invalid", method: TEST_METHOD_HAS_REQUIRED, req: &filestorev1.HasRequiredFilesRequest{UserId: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestCtx(t)
			h := &fakeHandler{resp: &emptypb.Empty{}}

			resp, err := Validate()(ctx, tt.req, &grpc.UnaryServerInfo{FullMethod: tt.method}, h.handle)

			if !tt.wantCalled {
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				assert.Nil(t, resp)
				assert.Empty(t, h.calls)
				return
			}
			require.NoError(t, err)
			assert.Same(t, h.resp, resp)
			assert.Len(t, h.calls, 1)
		})
	}
}

func TestValidate_Invariant_KnownFileTypesPass(t *testing.T) {
	tests := []struct {
		name     string
		fileType filestorev1.FileType
	}{
		{name: "passport", fileType: filestorev1.FileType_FILE_TYPE_PASSPORT},
		{name: "selfie", fileType: filestorev1.FileType_FILE_TYPE_SELFIE},
		{name: "proof of address", fileType: filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: tt.fileType, Filename: "doc.pdf"}
			runValidate(t, TEST_METHOD_INIT_UPLOAD, req, nil)
		})
	}
}
