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
