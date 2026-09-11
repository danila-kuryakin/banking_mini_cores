package grpc

import (
	"log/slog"

	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

type KycServer struct {
	kycv1.UnimplementedKycServiceServer
	log *slog.Logger
}

func NewKycServer(log *slog.Logger) *KycServer {
	return &KycServer{log: log}
}
