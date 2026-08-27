package adapters

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/kyc/v1"
)

type KycServer struct {
	kycv1.UnimplementedKycServiceServer
	repo *repository.Repository
	// events может быть nil - это штатный режим работы без Kafka,
	// методы продюсера в таком случае ничего не делают.
	events *kafka.Producer
	log    *slog.Logger
}

func NewKyc(db *pgxpool.Pool, events *kafka.Producer, log *slog.Logger) *KycServer {
	return &KycServer{
		repo:   repository.NewRepository(db),
		events: events,
		log:    log,
	}
}

func (s *KycServer) SubmitApplication(ctx context.Context, in *kycv1.SubmitApplicationRequest) (*kycv1.SubmitApplicationResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *KycServer) GetApplication(ctx context.Context, in *kycv1.GetApplicationRequest) (*kycv1.GetApplicationResponse, error) {
	applicationId := in.ApplicationId

	return &kycv1.GetApplicationResponse{
		Application: &kycv1.Application{
			ApplicationId: applicationId,
			CustomerId:    "",
			Status:        kycv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED,
			CreatedAt:     timestamppb.New(time.Now().UTC()),
		},
		Events: nil,
	}, s.repo.Kyc.GetApplication()
}
func (s *KycServer) ListApplications(ctx context.Context, in *kycv1.ListApplicationsRequest) (*kycv1.ListApplicationsResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *KycServer) ClaimApplication(ctx context.Context, in *kycv1.ClaimApplicationRequest) (*kycv1.ClaimApplicationResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *KycServer) ApproveApplication(ctx context.Context, in *kycv1.ApproveApplicationRequest) (*kycv1.ApproveApplicationResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *KycServer) RejectApplication(ctx context.Context, in *kycv1.RejectApplicationRequest) (*kycv1.RejectApplicationResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *KycServer) RequestMoreDocuments(ctx context.Context, in *kycv1.RequestMoreDocumentsRequest) (*kycv1.RequestMoreDocumentsResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
