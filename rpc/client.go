package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sudharma-networks/sudharma/transactions"
)

const DefaultClientURL = "http://127.0.0.1:18545"

type Client struct {
	baseURL string
	http    *http.Client
}

type NodeStatus struct {
	Network      string `json:"network"`
	Coin         string `json:"coin"`
	Symbol       string `json:"symbol"`
	NodeID       string `json:"node_id"`
	P2PAddress   string `json:"p2p_address"`
	Height       uint64 `json:"height"`
	TipHash      string `json:"tip_hash"`
	TotalWork    string `json:"total_work"`
	Peers        int    `json:"peers"`
	Mempool      int    `json:"mempool"`
	IssuedSupply uint64 `json:"issued_supply"`
}

type AccountInfo struct {
	Address        string `json:"address"`
	Balance        uint64 `json:"balance"`
	ConfirmedNonce uint64 `json:"confirmed_nonce"`
	NextNonce      uint64 `json:"next_nonce"`
}

type SubmitResult struct {
	TransactionID string `json:"transaction_id"`
	RelayedPeers  int    `json:"relayed_peers"`
	Accepted      bool   `json:"accepted"`
}

type TransactionStatus struct {
	Transaction   *transactions.Transaction `json:"transaction"`
	Status        string                    `json:"status"`
	BlockHeight   *uint64                   `json:"block_height,omitempty"`
	BlockHash     string                    `json:"block_hash,omitempty"`
	Confirmations uint64                    `json:"confirmations"`
}

type RPCError struct {
	StatusCode int
	Message    string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error (%d): %s", e.StatusCode, e.Message)
}

func NewClient(baseURL string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultClientURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid RPC URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("RPC URL must use http or https")
	}
	return &Client{
		baseURL: strings.TrimRight(parsed.String(), "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *Client) Status(ctx context.Context) (*NodeStatus, error) {
	var result NodeStatus
	if err := c.doJSON(ctx, http.MethodGet, "/v1/status", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Account(ctx context.Context, address string) (*AccountInfo, error) {
	if strings.TrimSpace(address) == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}
	var result AccountInfo
	if err := c.doJSON(ctx, http.MethodGet, "/v1/accounts/"+url.PathEscape(address), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SubmitTransaction(ctx context.Context, tx *transactions.Transaction) (*SubmitResult, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	var result SubmitResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/transactions", tx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Transaction(ctx context.Context, txID string) (*TransactionStatus, error) {
	if len(txID) != 64 || !isLowerHex(txID) {
		return nil, fmt.Errorf("invalid transaction ID")
	}
	var result TransactionStatus
	if err := c.doJSON(ctx, http.MethodGet, "/v1/transactions/"+txID, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestValue, responseValue any) error {
	if c == nil || c.http == nil {
		return fmt.Errorf("RPC client is unavailable")
	}
	var body io.Reader
	if requestValue != nil {
		encoded, err := json.Marshal(requestValue)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if requestValue != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, 4<<20)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var rpcErr errorResponse
		if err := json.NewDecoder(limited).Decode(&rpcErr); err != nil || rpcErr.Error == "" {
			rpcErr.Error = resp.Status
		}
		return &RPCError{StatusCode: resp.StatusCode, Message: rpcErr.Error}
	}
	if responseValue == nil {
		return nil
	}
	if err := json.NewDecoder(limited).Decode(responseValue); err != nil {
		return fmt.Errorf("invalid RPC response: %w", err)
	}
	return nil
}
