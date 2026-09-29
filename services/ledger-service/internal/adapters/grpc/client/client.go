package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/account/v1"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/antifraud/v1"
)

type Clients struct {
	Accounts interfaces.Accounts
	Fraud    interfaces.FraudChecker

	conns []*grpc.ClientConn
}

func NewClients(cfg *config.Config) (*Clients, error) {
	c := &Clients{}

	accountConn, err := c.dial(cfg.Upstreams.Account)
	if err != nil {
		return nil, c.closeOnError(err)
	}

	c.Accounts = NewAccountClient(accountv1.NewAccountServiceClient(accountConn))

	antifraudConn, err := c.dial(cfg.Upstreams.Antifraud)
	if err != nil {
		return nil, c.closeOnError(err)
	}

	c.Fraud = NewAntifraudClient(antifraudv1.NewAntifraudServiceClient(antifraudConn))

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
