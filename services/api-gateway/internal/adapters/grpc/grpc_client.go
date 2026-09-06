package grpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/auth/v1"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
)

type GRPCClients struct {
	Auth     authv1.AuthServiceClient
	Customer customerv1.CustomerServiceClient

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
