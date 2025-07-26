package client

import (
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
)

// Client wraps the gRPC client connection
type Client struct {
	conn          *grpc.ClientConn
	runnerService gradv1.RunnerServiceClient
}

// Config holds client configuration
type Config struct {
	ServerAddress string
	Timeout       time.Duration
}

// DefaultConfig returns default client configuration
func DefaultConfig() *Config {
	serverAddr := os.Getenv("GRAD_SERVER")
	if serverAddr == "" {
		serverAddr = "localhost:9090"
	}

	return &Config{
		ServerAddress: serverAddr,
		Timeout:       30 * time.Second,
	}
}

// NewClient creates a new gRPC client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	conn, err := grpc.NewClient(cfg.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to server %s: %w", cfg.ServerAddress, err)
	}

	return &Client{
		conn:          conn,
		runnerService: gradv1.NewRunnerServiceClient(conn),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RunnerService returns the runner service client
func (c *Client) RunnerService() gradv1.RunnerServiceClient {
	return c.runnerService
}