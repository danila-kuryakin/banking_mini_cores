package grpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/account/v1"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/antifraud/v1"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/auth/v1"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/filestore/v1"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/kyc/v1"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/ledger/v1"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/notification/v1"
)

type GRPCClients struct {
	Auth         authv1.AuthServiceClient
	Customer     customerv1.CustomerServiceClient
	Filestore    filestorev1.FileServiceClient
	Kyc          kycv1.KycServiceClient
	Account      accountv1.AccountServiceClient
	Ledger       ledgerv1.LedgerServiceClient
	Antifraud    antifraudv1.AntifraudServiceClient
	Notification notificationv1.NotificationServiceClient

	conns []*grpc.ClientConn // Для удобного закрытия коннектов клиентов
}

func NewGRPCClients(cfg *config.Config) (*GRPCClients, error) {
	c := &GRPCClients{}

	authConn, err := c.dial(cfg.Upstreams.Auth)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Auth = authv1.NewAuthServiceClient(authConn)

	customerConn, err := c.dial(cfg.Upstreams.Customer)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Customer = customerv1.NewCustomerServiceClient(customerConn)

	filestoreConn, err := c.dial(cfg.Upstreams.Filestore)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Filestore = filestorev1.NewFileServiceClient(filestoreConn)

	kycConn, err := c.dial(cfg.Upstreams.KYC)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Kyc = kycv1.NewKycServiceClient(kycConn)

	accountConn, err := c.dial(cfg.Upstreams.Account)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Account = accountv1.NewAccountServiceClient(accountConn)

	ledgerConn, err := c.dial(cfg.Upstreams.Ledger)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Ledger = ledgerv1.NewLedgerServiceClient(ledgerConn)

	antifraudConn, err := c.dial(cfg.Upstreams.Antifraud)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Antifraud = antifraudv1.NewAntifraudServiceClient(antifraudConn)

	notificationConn, err := c.dial(cfg.Upstreams.Notification)
	if err != nil {
		return nil, c.closeOnError(err)
	}
	c.Notification = notificationv1.NewNotificationServiceClient(notificationConn)

	return c, nil
}

func (c *GRPCClients) dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c.conns = append(c.conns, conn)

	return conn, nil
}

func (c *GRPCClients) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}

func (c *GRPCClients) closeOnError(err error) error {
	c.Close()

	return err
}
