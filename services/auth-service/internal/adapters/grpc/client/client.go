package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/customer/v1"
)

type Clients struct {
	Customer interfaces.CustomerProfiles

	conns []*grpc.ClientConn
}

func NewClients(cfg *config.Config) (*Clients, error) {
	c := &Clients{}

	customerConn, err := c.dial(cfg.Upstreams.Customer)
	if err != nil {
		return nil, c.closeOnError(err)
	}

	c.Customer = NewCustomerClient(customerv1.NewCustomerServiceClient(customerConn))

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
