package handler

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain/models"
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
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				dto.ErrorResponse{Message: err.Error()},
			)

			return
		}

		if !out.Valid {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrAccessTokenNotValid.Error()},
			)

			return
		}

		role, err := normalizeRole(out.Role)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				dto.ErrorResponse{Message: domain.ErrRoleNotAllowed.Error()},
			)

			return
		}

		c.Request = c.Request.WithContext(metadata.AppendToOutgoingContext(
			c.Request.Context(),
			domain.AUTHORIZATION_METADATA,
			domain.BEARER_PREFIX+accessToken,
		))

		c.Set(domain.ACTOR_KEY, models.Actor{
			ID:   out.ID,
			Role: role,
		})

		c.Next()
	}
}

// RequireRole пропускает дальше только перечисленные роли. Ставится после
// RequireAuth - актора берёт из контекста, сам токен не разбирает.
func (m *Middleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrAuthenticationNeeded.Error()},
			)

			return
		}

		if !slices.Contains(roles, actor.Role) {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				dto.ErrorResponse{Message: domain.ErrRoleNotAllowed.Error()},
			)

			return
		}

		c.Next()
	}
}

// RequireSelfOrRole пропускает владельца ресурса и перечисленные роли. Без неё
// любой авторизованный клиент читал бы и перезаписывал чужой профиль, просто
// подставив чужой идентификатор в путь: ниже по стеку владение никто не
// проверяет.
func (m *Middleware) RequireSelfOrRole(param string, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFrom(c)
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				dto.ErrorResponse{Message: domain.ErrAuthenticationNeeded.Error()},
			)

			return
		}

		target := c.Param(param)
		if target == domain.SELF_ALIAS || target == actor.ID || slices.Contains(roles, actor.Role) {
			c.Next()

			return
		}

		c.AbortWithStatusJSON(
			http.StatusForbidden,
			dto.ErrorResponse{Message: domain.ErrForeignCustomer.Error()},
		)
	}
}

func ActorFrom(c *gin.Context) (models.Actor, bool) {
	value, ok := c.Get(domain.ACTOR_KEY)
	if !ok {
		return models.Actor{}, false
	}

	actor, ok := value.(models.Actor)

	return actor, ok
}

func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader(domain.AUTHORIZATION_HEADER)
	if !strings.HasPrefix(header, domain.BEARER_PREFIX) {
		return "", false
	}

	accessToken := strings.TrimSpace(strings.TrimPrefix(header, domain.BEARER_PREFIX))

	return accessToken, accessToken != ""
}

// normalizeRole приводит claim "role" из access-токена к доменному имени роли.
// Auth-service кладёт туда значение в нижнем регистре ("client"), но имена
// значений enum'а common.v1.Role выглядят как "ROLE_CLIENT" - принимаем обе
// формы, чтобы смена формата claim'а не ломала авторизацию молча.
func normalizeRole(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.TrimPrefix(name, strings.ToLower(domain.ROLE_ENUM_PREFIX))

	switch name {
	case domain.ROLE_CLIENT, domain.ROLE_OFFICER, domain.ROLE_ADMIN:
		return name, nil
	default:
		return "", domain.ErrRoleNotAllowed
	}
}
