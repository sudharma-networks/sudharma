package gpuminer

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

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid mining RPC URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("mining RPC URL must use http or https")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		baseURL: trimmed,
		http:    &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) GetWork(ctx context.Context, address string) (Work, error) {
	if err := ValidateRewardAddress(address); err != nil {
		return Work{}, err
	}
	body, err := json.Marshal(map[string]string{"address": address})
	if err != nil {
		return Work{}, err
	}
	raw, err := c.do(ctx, http.MethodPost, "/v1/mining/work", body)
	if err != nil {
		if isMissingRoute(err) {
			raw, err = c.do(ctx, http.MethodGet, "/v1/mining/work?address="+url.QueryEscape(address), nil)
		}
		if err != nil {
			return Work{}, err
		}
	}
	return WorkFromJSON(raw)
}

func (c *Client) Submit(ctx context.Context, work Work, nonce uint64) (SubmitResult, error) {
	if err := params.ValidateProductionMiningAlgorithm(work.Algorithm); err != nil {
		return SubmitResult{}, err
	}
	payload, err := json.Marshal(Solution{
		WorkID:        work.WorkID,
		Nonce:         nonce,
		Algorithm:     work.Algorithm,
		Version:       work.Version,
		Height:        work.Height,
		Difficulty:    work.Difficulty,
		Target:        work.Target,
		HeaderPrefix:  work.HeaderPrefix,
		RewardAddress: work.RewardAddress,
		CacheNodes:    work.CacheNodes,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	return c.decodeSubmit(ctx, payload)
}

func (c *Client) SubmitBlock(ctx context.Context, block *blockchain.Block) (SubmitResult, error) {
	if block == nil {
		return SubmitResult{}, fmt.Errorf("missing block")
	}
	payload, err := json.Marshal(block)
	if err != nil {
		return SubmitResult{}, err
	}
	return c.decodeSubmit(ctx, payload)
}

func (c *Client) decodeSubmit(ctx context.Context, payload []byte) (SubmitResult, error) {
	raw, err := c.do(ctx, http.MethodPost, "/v1/mining/submit", payload)
	if err != nil {
		return SubmitResult{}, err
	}
	var result SubmitResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return SubmitResult{}, fmt.Errorf("invalid mining submit JSON: %w", err)
	}
	if result.Accepted && result.Status == "" {
		result.Status = "accepted"
	}
	return result, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if c == nil || c.http == nil {
		return nil, fmt.Errorf("mining RPC client is unavailable")
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mining RPC request failed: %w", err)
	}
	defer resp.Body.Close()
	limited, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var rpcErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(limited, &rpcErr) != nil || rpcErr.Error == "" {
			rpcErr.Error = strings.TrimSpace(string(limited))
			if rpcErr.Error == "" {
				rpcErr.Error = resp.Status
			}
		}
		return nil, fmt.Errorf("mining RPC %d: %s", resp.StatusCode, rpcErr.Error)
	}
	return limited, nil
}

func isMissingRoute(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "mining rpc 404") ||
		strings.Contains(msg, "mining rpc 405") ||
		strings.Contains(msg, "route not found")
}
