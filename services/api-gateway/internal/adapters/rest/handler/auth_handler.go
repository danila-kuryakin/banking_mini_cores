package handler

import (
	"net/http"

	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/auth/v1"
	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
)

type AuthHandler struct {
	service *service.Service
}

func NewAuthHandler(service *service.Service) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// Register godoc
//
//	@Summary		Регистрация пользователя
//	@Description	Создаёт учётную запись клиента. Пароль — от 8 до 72 символов.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest		true	"Email и пароль"
//	@Success		201		{object}	dto.RegisterResponse	"Пользователь создан"
//	@Failure		400		{object}	dto.ErrorResponse		"Тело запроса не прошло валидацию"
//	@Failure		500		{object}	dto.ErrorResponse		"Ошибка auth-service"
//	@Router			/auth/register [post]
func (h AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	user, err := h.service.Auth.Register(c.Request.Context(), &authv1.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}
	if user == nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "empty response"})
		return
	}
	c.JSON(http.StatusCreated, dto.RegisterResponse{
		ID:        user.Id,
		Email:     user.Email,
		Role:      user.Role.String(),
		CreatedAt: user.CreatedAt.AsTime(),
	})
}

// Login godoc
//
//	@Summary		Вход по email и паролю
//	@Description	Возвращает данные пользователя и пару токенов. Access-токен передаётся дальше в заголовке Authorization.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Email и пароль"
//	@Success		200		{object}	dto.LoginResponse	"Пользователь и пара токенов"
//	@Failure		400		{object}	dto.ErrorResponse	"Тело запроса не прошло валидацию"
//	@Failure		500		{object}	dto.ErrorResponse	"Неверные учётные данные или ошибка auth-service"
//	@Router			/auth/login [post]
func (h AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	resp, err := h.service.Auth.Login(c.Request.Context(), &authv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{
		User: dto.User{
			ID:        resp.User.Id,
			Email:     resp.User.Email,
			Role:      resp.User.Role.String(),
			CreatedAt: resp.User.CreatedAt.AsTime(),
		},
		Tokens: dto.TokenPair{
			AccessToken:      resp.Tokens.AccessToken,
			RefreshToken:     resp.Tokens.RefreshToken,
			AccessExpiresAt:  resp.Tokens.AccessExpiresAt.AsTime(),
			RefreshExpiresAt: resp.Tokens.RefreshExpiresAt.AsTime(),
		},
	})
}

// RefreshToken godoc
//
//	@Summary		Обновление пары токенов
//	@Description	Выдаёт новую пару токенов по refresh-токену. Старый refresh-токен после обмена недействителен.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RefreshRequest	true	"Refresh-токен"
//	@Success		200		{object}	dto.TokenPair		"Новая пара токенов"
//	@Failure		400		{object}	dto.ErrorResponse	"Тело запроса не прошло валидацию"
//	@Failure		500		{object}	dto.ErrorResponse	"Refresh-токен просрочен, отозван или ошибка auth-service"
//	@Router			/auth/refresh [post]
func (h AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	out, err := h.service.Auth.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}

	c.JSON(http.StatusOK, dto.TokenPair{
		AccessToken:      out.AccessToken,
		RefreshToken:     out.RefreshToken,
		AccessExpiresAt:  out.AccessExpiresAt.AsTime(),
		RefreshExpiresAt: out.RefreshExpiresAt.AsTime(),
	})
}

// Logout godoc
//
//	@Summary		Выход из сессии
//	@Description	Отзывает refresh-токен. Ранее выданный access-токен продолжает действовать до истечения срока.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	dto.LogoutRequest	true	"Refresh-токен"
//	@Success		204		"Сессия завершена, тело ответа пустое"
//	@Failure		400		{object}	dto.ErrorResponse	"Тело запроса не прошло валидацию"
//	@Failure		500		{object}	dto.ErrorResponse	"Ошибка auth-service"
//	@Router			/auth/logout [post]
func (h AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	if err := h.service.Auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ValidateToken godoc
//
//	@Summary		Проверка access-токена
//	@Description	Служебный метод: сообщает, валиден ли токен, и возвращает идентификатор владельца, роль и срок действия.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ValidateTokenRequest	true	"Access-токен"
//	@Success		200		{object}	dto.ValidateTokenResponse	"Результат проверки; valid=false для просроченного токена"
//	@Failure		400		{object}	dto.ErrorResponse			"Тело запроса не прошло валидацию"
//	@Failure		500		{object}	dto.ErrorResponse			"Ошибка auth-service"
//	@Router			/auth/validate [post]
func (h AuthHandler) ValidateToken(c *gin.Context) {
	var req dto.ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	out, err := h.service.Auth.ValidateToken(c.Request.Context(), req.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}

	c.JSON(http.StatusOK, dto.ValidateTokenResponse{
		Valid:     out.Valid,
		ID:        out.ID,
		Role:      out.Role,
		ExpiresAt: out.ExpiresAt,
	})
}

// CreateOfficers godoc
//
//	@Summary		Создание сотрудника
//	@Description	Заводит учётную запись с ролью officer. Доступно только роли admin.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreateOfficerRequest	true	"Email и пароль сотрудника"
//	@Success		201		{object}	dto.User					"Сотрудник создан"
//	@Failure		400		{object}	dto.ErrorResponse			"Тело запроса не прошло валидацию"
//	@Failure		401		{object}	dto.ErrorResponse			"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse			"Роль не admin"
//	@Failure		500		{object}	dto.ErrorResponse			"Ошибка auth-service"
//	@Router			/auth/officers [post]
func (h AuthHandler) CreateOfficers(c *gin.Context) {
	var req dto.CreateOfficerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	out, err := h.service.Auth.CreateOfficers(c.Request.Context(), &authv1.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}

	c.JSON(http.StatusCreated, dto.User{
		ID:        out.Id,
		Email:     out.Email,
		Role:      out.Role.String(),
		CreatedAt: out.CreatedAt.AsTime(),
	})
}
