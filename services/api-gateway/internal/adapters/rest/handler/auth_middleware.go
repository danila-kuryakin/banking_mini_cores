package handler

import (
	"net/http"
	"slices"
	"strings"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"

	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/common/v1"
)

type Middleware struct {
	service *service.Service
}

func NewMiddleware(service *service.Service) *Middleware {
	return &Middleware{
		service: service,
	}
}

func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, ok := bearerToken(c)
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrBearerTokenRequired.Error()},
			)
			return
		}

		out, err := m.service.Auth.ValidateToken(c.Request.Context(), accessToken)
		if err != nil {
			c.Abort()
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})

			return
		}

		if !out.Valid {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrAccessTokenNotValid.Error()},
			)

			return
		}

		c.Request = c.Request.WithContext(metadata.AppendToOutgoingContext(
			c.Request.Context(),
			domain.AUTHORIZATION_METADATA,
			domain.BEARER_PREFIX+accessToken,
		))

		role, err := parseRole(out.Role)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrAccessTokenNotValid.Error()},
			)

			return
		}

		c.Set(domain.ACTOR_KEY, commonv1.Actor{
			Id:   out.ID,
			Role: role,
		})
		c.Next()
	}
}

func (m *Middleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := c.Get(domain.ACTOR_KEY)
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrAuthenticationNeeded.Error()},
			)
			return
		}

		if !slices.Contains(roles, actor.(commonv1.Actor).Role.String()) {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				dto.ErrorResponse{Message: domain.ErrRoleNotAllowed.Error()},
			)

			return
		}

		c.Next()
	}
}

func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader(domain.AUTHORIZATION_HEADER)
	if !strings.HasPrefix(header, domain.BEARER_PREFIX) {
		return "", false
	}

	accessToken := strings.TrimSpace(strings.TrimPrefix(header, domain.BEARER_PREFIX))

	return accessToken, accessToken != ""
}

func parseRole(s string) (commonv1.Role, error) {
	v, ok := commonv1.Role_value[strings.ToUpper(strings.TrimSpace(s))]
	if !ok || commonv1.Role(v) == commonv1.Role_ROLE_UNSPECIFIED {
		return commonv1.Role_ROLE_UNSPECIFIED, domain.ErrRoleNotAllowed
	}

	return commonv1.Role(v), nil
}
