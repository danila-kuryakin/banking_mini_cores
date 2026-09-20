package service

import (
	"context"
	"time"

	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/kyc/v1"
)

type KycService struct {
	kycClient kycv1.KycServiceClient
	timeout   time.Duration
}

func NewKycService(kycCli kycv1.KycServiceClient, timeout time.Duration) *KycService {
	return &KycService{
		kycClient: kycCli,
		timeout:   timeout,
	}
}

func (s KycService) SubmitApplication(ctx context.Context, userID string) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.SubmitApplication(ctx, &kycv1.SubmitApplicationRequest{UserId: userID})
}

func (s KycService) GetByUser(ctx context.Context, userID string) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.GetApplicationByUserID(ctx, &kycv1.GetApplicationByUserIDRequest{UserId: userID})
}

func (s KycService) GetByID(ctx context.Context, applicationID string) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.GetApplicationByID(ctx, &kycv1.GetApplicationByIDRequest{ApplicationId: applicationID})
}

func (s KycService) ListApplications(ctx context.Context, in *kycv1.ListApplicationsRequest) ([]*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.kycClient.ListApplications(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Applications, nil
}

func (s KycService) Claim(ctx context.Context, applicationID, officerID string) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.ClaimApplication(ctx, &kycv1.ClaimApplicationRequest{
		ApplicationId: applicationID,
		OfficerId:     officerID,
	})
}

func (s KycService) Approve(ctx context.Context, applicationID, officerID, comment string) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.ApproveApplication(ctx, &kycv1.ApproveApplicationRequest{
		ApplicationId: applicationID,
		OfficerId:     officerID,
		Comment:       comment,
	})
}

func (s KycService) Reject(ctx context.Context, applicationID, officerID, reason string) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.RejectApplication(ctx, &kycv1.RejectApplicationRequest{
		ApplicationId: applicationID,
		OfficerId:     officerID,
		Reason:        reason,
	})
}

func (s KycService) RequestDocuments(
	ctx context.Context,
	applicationID, officerID string,
	types []kycv1.DocumentType,
	comment string,
) (*kycv1.Application, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.RequestMoreDocuments(ctx, &kycv1.RequestMoreDocumentsRequest{
		ApplicationId: applicationID,
		OfficerId:     officerID,
		Types:         types,
		Comment:       comment,
	})
}

func (s KycService) AddSanctionEntry(ctx context.Context, in *kycv1.AddSanctionEntryRequest) (*kycv1.SanctionEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.kycClient.AddSanctionEntry(ctx, in)
}

func (s KycService) ListSanctions(ctx context.Context, limit, offset int32) ([]*kycv1.SanctionEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.kycClient.ListSanctions(ctx, &kycv1.ListSanctionsRequest{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	return listed.Entries, nil
}
