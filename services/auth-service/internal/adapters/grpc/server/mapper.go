package server

import (
	"errors"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AuthServer) credentialsError(err error, message string) error {
	switch {
	case errors.Is(err, domain.ErrEmailRequired), errors.Is(err, domain.ErrPasswordRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrEmailTaken):
		return domain.ErrEmailAlreadyExists
	default:
		s.log.Error(message, "error", err)

		return status.Error(codes.Internal, message)
	}
}

func roleToProto(role models.Role) commonv1.Role {
	switch role {
	case domain.ROLE_CLIENT:
		return commonv1.Role_ROLE_CLIENT
	case domain.ROLE_OFFICER:
		return commonv1.Role_ROLE_OFFICER
	case domain.ROLE_ADMIN:
		return commonv1.Role_ROLE_ADMIN
	default:
		return commonv1.Role_ROLE_UNSPECIFIED
	}
}
