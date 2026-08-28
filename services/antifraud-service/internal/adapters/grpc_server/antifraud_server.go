package grpc_server

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/adapters/repository"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

type AntifraudServer struct {
	antifraudv1.UnimplementedAntifraudServiceServer
	repo *repository.Repository
	log  *slog.Logger
}

func NewAntifraud(db *pgxpool.Pool, log *slog.Logger) *AntifraudServer {
	return &AntifraudServer{
		repo: repository.NewRepository(db),
		log:  log,
	}
}

func (s *AntifraudServer) CheckTransfer(ctx context.Context, in *antifraudv1.CheckTransferRequest) (*antifraudv1.CheckTransferResponse, error) {

	idT := in.TransactionId

	return &antifraudv1.CheckTransferResponse{
		CheckId:        idT,
		Verdict:        antifraudv1.Verdict_VERDICT_UNSPECIFIED,
		TriggeredRules: nil,
	}, s.repo.Antifraud.CheckTransfer()
}
func (s *AntifraudServer) ListRules(ctx context.Context, in *antifraudv1.ListRulesRequest) (*antifraudv1.ListRulesResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AntifraudServer) UpdateRule(ctx context.Context, in *antifraudv1.UpdateRuleRequest) (*antifraudv1.UpdateRuleResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *AntifraudServer) ListChecks(ctx context.Context, in *antifraudv1.ListChecksRequest) (*antifraudv1.ListChecksResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
