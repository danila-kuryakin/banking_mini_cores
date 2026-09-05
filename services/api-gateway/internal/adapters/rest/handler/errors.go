package handler

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrInvalidCustomerStatus = status.Error(codes.InvalidArgument, "status is not a valid customer status")
