package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
)

// writeError переводит ошибку нижнего сервиса в HTTP-ответ. Сервисы отдают
// ошибки как gRPC-статусы, и без этого перевода клиент получал бы 500 на любую
// причину - от "клиент не найден" до "профиль уже заполнен".
func writeError(c *gin.Context, err error) {
	_ = c.Error(err)

	st, ok := status.FromError(err)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})

		return
	}

	c.AbortWithStatusJSON(httpStatus(st.Code()), dto.ErrorResponse{Message: st.Message()})
}

// writeBindError отвечает на непрочитанное тело или query-строку. Это ошибка
// самого запроса, до нижних сервисов дело не дошло.
func writeBindError(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
}

func httpStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
