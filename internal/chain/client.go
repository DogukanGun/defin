package chain

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	http *ethclient.Client
	ws   *ethclient.Client
	cfg  ClientConfig
	log  *slog.Logger
	mu   sync.RWMutex
}

type ClientConfig struct {
	Name        string
	ChainID     int64
	RPCHTTP     string
	RPCWS       string
	Multicall3  string
}

func NewClient(ctx context.Context, cfg ClientConfig, log *slog.Logger) (*Client, error) {
	c := &Client{cfg: cfg, log: log}

	if cfg.RPCHTTP != "" {
		httpClient, err := ethclient.DialContext(ctx, cfg.RPCHTTP)
		if err != nil {
			return nil, fmt.Errorf("dial http %s: %w", cfg.Name, err)
		}
		c.http = httpClient
	}

	if cfg.RPCWS != "" {
		wsClient, err := ethclient.DialContext(ctx, cfg.RPCWS)
		if err != nil {
			return nil, fmt.Errorf("dial ws %s: %w", cfg.Name, err)
		}
		c.ws = wsClient
	}

	if err := c.healthCheck(ctx); err != nil {
		return nil, fmt.Errorf("health check %s: %w", cfg.Name, err)
	}

	log.Info("chain client connected", "chain", cfg.Name, "chain_id", cfg.ChainID)
	return c, nil
}

func (c *Client) healthCheck(ctx context.Context) error {
	cl := c.HTTP()
	if cl == nil {
		return fmt.Errorf("no client available")
	}

	chainID, err := cl.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("get chain id: %w", err)
	}
	if chainID.Int64() != c.cfg.ChainID {
		return fmt.Errorf("chain id mismatch: expected %d, got %d", c.cfg.ChainID, chainID.Int64())
	}
	return nil
}

func (c *Client) HTTP() *ethclient.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.http != nil {
		return c.http
	}
	return c.ws
}

func (c *Client) WS() *ethclient.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ws != nil {
		return c.ws
	}
	return c.http
}

func (c *Client) ChainID() int64 {
	return c.cfg.ChainID
}

func (c *Client) Name() string {
	return c.cfg.Name
}

func (c *Client) Multicall3Address() string {
	return c.cfg.Multicall3
}

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.HTTP().BlockNumber(ctx)
}

func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.HTTP().SuggestGasPrice(ctx)
}

func (c *Client) Close() {
	if c.http != nil {
		c.http.Close()
	}
	if c.ws != nil {
		c.ws.Close()
	}
}
