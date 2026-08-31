package gpuminer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// RPCClient is the mining RPC surface used by the GPU miner loop.
type RPCClient interface {
	GetWork(ctx context.Context, address string) (Work, error)
	Submit(ctx context.Context, work Work, nonce uint64) (SubmitResult, error)
	SubmitBlock(ctx context.Context, block *blockchain.Block) (SubmitResult, error)
	NetworkStatus(ctx context.Context) (NetworkStatus, error)
	Endpoint() string
}

// FailoverClient tries seed-1 then seed-2 (and optional public proxy) in the
// same order as deployment/testnet/public-rpc/lambda/upstream.mjs.
type FailoverClient struct {
	clients []*Client
}

func NewFailoverClient(endpoints []string, timeout time.Duration) (*FailoverClient, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("at least one mining RPC endpoint is required")
	}
	clients := make([]*Client, 0, len(endpoints))
	for _, endpoint := range endpoints {
		client, err := NewClient(endpoint, timeout)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return &FailoverClient{clients: clients}, nil
}

func (f *FailoverClient) Endpoint() string {
	if f == nil || len(f.clients) == 0 {
		return ""
	}
	if len(f.clients) == 1 {
		return f.clients[0].baseURL
	}
	parts := make([]string, 0, len(f.clients))
	for _, client := range f.clients {
		parts = append(parts, client.baseURL)
	}
	return strings.Join(parts, " -> ")
}

func (f *FailoverClient) NetworkStatus(ctx context.Context) (NetworkStatus, error) {
	return callWithFailover(f, func(client *Client) (NetworkStatus, error) {
		return client.NetworkStatus(ctx)
	})
}

func (f *FailoverClient) GetWork(ctx context.Context, address string) (Work, error) {
	return callWithFailover(f, func(client *Client) (Work, error) {
		return client.GetWork(ctx, address)
	})
}

func (f *FailoverClient) Submit(ctx context.Context, work Work, nonce uint64) (SubmitResult, error) {
	return callWithFailover(f, func(client *Client) (SubmitResult, error) {
		return client.Submit(ctx, work, nonce)
	})
}

func (f *FailoverClient) SubmitBlock(ctx context.Context, block *blockchain.Block) (SubmitResult, error) {
	return callWithFailover(f, func(client *Client) (SubmitResult, error) {
		return client.SubmitBlock(ctx, block)
	})
}

func callWithFailover[T any](f *FailoverClient, fn func(*Client) (T, error)) (T, error) {
	var zero T
	if f == nil || len(f.clients) == 0 {
		return zero, fmt.Errorf("mining RPC client is unavailable")
	}
	var lastErr error
	for i, client := range f.clients {
		result, err := fn(client)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if i+1 >= len(f.clients) || !isRetryableMiningRPCError(err) {
			break
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("mining RPC request failed")
	}
	return zero, lastErr
}

func isRetryableMiningRPCError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "mining rpc request failed") {
		return true
	}
	for _, code := range []string{"500", "502", "503", "504"} {
		if strings.Contains(msg, "mining rpc "+code) {
			return true
		}
	}
	return false
}

// Ensure Client and FailoverClient implement RPCClient.
var (
	_ RPCClient = (*Client)(nil)
	_ RPCClient = (*FailoverClient)(nil)
)

func (c *Client) Endpoint() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}
