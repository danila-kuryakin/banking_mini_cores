package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/customer/v1"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/filestore/v1"
)

type Clients struct {
	Customer  interfaces.CustomerProfiles
	Documents interfaces.Documents

	conns []*grpc.ClientConn
}

func NewClients(cfg *config.Config) (*Clients, error) {
	c := &Clients{}

	customerConn, err := c.dial(cfg.Upstreams.Customer)
	if err != nil {
		return nil, c.closeOnError(err)
	}

	// Два контракта одного сервиса: анкета читается через CustomerService,
	// статус двигается через CustomerStatusService.
	c.Customer = NewCustomerClient(
		customerv1.NewCustomerServiceClient(customerConn),
		customerv1.NewCustomerStatusServiceClient(customerConn),
	)

	filestoreConn, err := c.dial(cfg.Upstreams.Filestore)
	if err != nil {
		return nil, c.closeOnError(err)
	}

	c.Documents = NewFilestoreClient(filestorev1.NewFileServiceClient(filestoreConn))

	return c, nil
}

func (c *Clients) dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c.conns = append(c.conns, conn)

	return conn, nil
}

func (c *Clients) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}

func (c *Clients) closeOnError(err error) error {
	c.Close()

	return err
}
