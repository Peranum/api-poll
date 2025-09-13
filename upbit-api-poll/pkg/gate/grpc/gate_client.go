package grpc

import (
	"context"
	"fmt"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/gate/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GateClient struct {
	conn       *grpc.ClientConn
	gateClient proto.GateServiceClient
	config     config.GRPC
}

func NewGateClient(cfg config.GRPC) (*GateClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gate service: %w", err)
	}

	client := &GateClient{
		conn:       conn,
		gateClient: proto.NewGateServiceClient(conn),
		config:     cfg,
	}

	return client, nil
}

func (c *GateClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *GateClient) OpenOrder(ctx context.Context, ticker string) error {
	if ticker == "" {
		return fmt.Errorf("ticker cannot be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.CallTimeout)
	defer cancel()

	req := &proto.OpenOrderRequest{
		Ticker: ticker,
	}

	resp, err := c.gateClient.OpenOrder(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to open order: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to open order: %s", resp.Error)
	}

	return nil
}
