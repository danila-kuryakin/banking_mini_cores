package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/filestore/v1"
)

type FilestoreClient struct {
	files filestorev1.FileServiceClient
}

func NewFilestoreClient(files filestorev1.FileServiceClient) *FilestoreClient {
	return &FilestoreClient{files: files}
}

func (c *FilestoreClient) HasRequiredDocuments(ctx context.Context, userID uuid.UUID) (bool, []models.DocumentType, error) {
	resp, err := c.files.HasRequiredFiles(ctx, &filestorev1.HasRequiredFilesRequest{UserId: userID.String()})
	if err != nil {
		return false, nil, fmt.Errorf("has required files: %w", err)
	}

	missing := make([]models.DocumentType, 0, len(resp.Missing))

	for _, fileType := range resp.Missing {
		missing = append(missing, documentTypeFromProto(fileType))
	}

	return resp.Ok, missing, nil
}

func (c *FilestoreClient) ConfirmedTypes(ctx context.Context, userID uuid.UUID) ([]models.DocumentType, error) {
	resp, err := c.files.ListFiles(ctx, &filestorev1.ListFilesRequest{
		UserId: userID.String(),
		Limit:  domain.FILE_LIST_LIMIT,
	})
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	seen := make(map[models.DocumentType]struct{}, len(resp.Files))
	types := make([]models.DocumentType, 0, len(resp.Files))

	for _, file := range resp.Files {
		if file.Status != filestorev1.FileStatus_FILE_STATUS_CONFIRMED {
			continue
		}

		docType := documentTypeFromProto(file.Type)
		if !docType.IsValid() {
			continue
		}

		if _, ok := seen[docType]; ok {
			continue
		}

		seen[docType] = struct{}{}
		types = append(types, docType)
	}

	return types, nil
}
