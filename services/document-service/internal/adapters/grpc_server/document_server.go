package grpc_server

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/adapters/repository"
	documentv1 "github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/pb/gen/document/v1"
)

type DocumentServer struct {
	documentv1.UnimplementedDocumentServiceServer
	repo *repository.Repository
	log  *slog.Logger
}

func NewDocument(db *pgxpool.Pool, log *slog.Logger) *DocumentServer {
	return &DocumentServer{
		repo: repository.NewRepository(db),
		log:  log,
	}
}

func (s *DocumentServer) InitUpload(ctx context.Context, in *documentv1.InitUploadRequest) (*documentv1.InitUploadResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *DocumentServer) ConfirmUpload(ctx context.Context, in *documentv1.ConfirmUploadRequest) (*documentv1.ConfirmUploadResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *DocumentServer) GetDownloadUrl(ctx context.Context, in *documentv1.GetDownloadUrlRequest) (*documentv1.GetDownloadUrlResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
func (s *DocumentServer) ListDocuments(ctx context.Context, in *documentv1.ListDocumentsRequest) (*documentv1.ListDocumentsResponse, error) {

	return &documentv1.ListDocumentsResponse{
		Documents: []*documentv1.Document{
			{DocumentId: in.CustomerId},
		},
		Page: nil,
	}, s.repo.Document.ListDocuments()
}
func (s *DocumentServer) HasRequiredDocuments(ctx context.Context, in *documentv1.HasRequiredDocumentsRequest) (*documentv1.HasRequiredDocumentsResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}
