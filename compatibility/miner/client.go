package miner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const maxResponseBytes = int64(64 << 10)

// Work mirrors the immutable external mining work template. Nonce is the only
// miner-controlled consensus field and is therefore intentionally absent.
type Work struct {
	WorkID        string `json:"work_id"`
	Algorithm     string `json:"algorithm"`
	Version       uint32 `json:"version"`
	Height        uint64 `json:"height"`
	Difficulty    uint32 `json:"difficulty"`
	Target        string `json:"target"`
	HeaderPrefix  string `json:"header_prefix"`
	RewardAddress string `json:"reward_address"`
}

type Solution struct {
	WorkID        string `json:"work_id"`
	Nonce         uint64 `json:"nonce"`
	Algorithm     string `json:"algorithm"`
	Version       uint32 `json:"version"`
	Height        uint64 `json:"height"`
	Difficulty    uint32 `json:"difficulty"`
	Target        string `json:"target"`
	HeaderPrefix  string `json:"header_prefix"`
	RewardAddress string `json:"reward_address"`
}

type SubmitResult struct {
	Status string `json:"status"`
}

type Stats struct {
	Accepted uint64
	Rejected uint64
	Stale    uint64
}

type Verifier func(work Work, nonce uint64) bool

type Client struct {
	base *url.URL
	http *http.Client

	mu         sync.RWMutex
	workID     string
	work       Work
	generation uint64
	stats      Stats
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid mining base URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported mining URL scheme")
	}
	if u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("mining base URL must not include a path, query, or fragment")
	}
	u.Path = ""
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{base: u, http: httpClient}, nil
}

func (c *Client) Poll(ctx context.Context) (Work, uint64, error) {
	var work Work
	if err := c.doJSON(ctx, http.MethodGet, "/v1/mining/work", nil, &work); err != nil {
		return Work{}, 0, err
	}
	if err := validateWork(work); err != nil {
		return Work{}, 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if work.WorkID == c.workID && c.workID != "" && work != c.work {
		return Work{}, 0, errors.New("mining work_id reused with mutated immutable template")
	}
	if work.WorkID != c.workID {
		c.generation++
		c.workID = work.WorkID
		c.work = work
	}
	return work, c.generation, nil
}

func (c *Client) IsCurrent(workID string, generation uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return workID != "" && c.workID == workID && c.generation == generation
}

func (c *Client) isCurrentWork(work Work, generation uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return work.WorkID != "" && c.workID == work.WorkID && c.generation == generation && c.work == work
}

func (c *Client) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *Client) recordSubmitStatus(status string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch status {
	case "accepted":
		c.stats.Accepted++
	case "stale":
		c.stats.Stale++
	case "invalid", "mutated":
		c.stats.Rejected++
	default:
		return false
	}
	return true
}

func (c *Client) SubmitVerified(ctx context.Context, work Work, generation, nonce uint64, verifier Verifier) (SubmitResult, error) {
	if !c.isCurrentWork(work, generation) {
		return SubmitResult{}, errors.New("stale or mutated mining work")
	}
	if verifier == nil || !verifier(work, nonce) {
		return SubmitResult{}, errors.New("candidate failed independent host verification")
	}
	if !c.isCurrentWork(work, generation) {
		return SubmitResult{}, errors.New("mining work became stale or mutated before submission")
	}

	solution := Solution{
		WorkID: work.WorkID, Nonce: nonce, Algorithm: work.Algorithm,
		Version: work.Version, Height: work.Height, Difficulty: work.Difficulty,
		Target: work.Target, HeaderPrefix: work.HeaderPrefix, RewardAddress: work.RewardAddress,
	}
	var result SubmitResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/mining/submit", solution, &result); err != nil {
		return SubmitResult{}, err
	}
	if strings.TrimSpace(result.Status) == "" {
		return SubmitResult{}, errors.New("mining submit response missing status")
	}
	if !c.recordSubmitStatus(result.Status) {
		return SubmitResult{}, fmt.Errorf("unknown mining submit status %q", result.Status)
	}
	return result, nil
}

func validateWork(work Work) error {
	if work.WorkID == "" || work.Algorithm != "sudharma-gpupow-v1" || work.Version != 2 ||
		work.Height == 0 && work.HeaderPrefix == "" || work.Difficulty == 0 || work.Target == "" ||
		work.HeaderPrefix == "" || strings.TrimSpace(work.RewardAddress) == "" {
		return errors.New("invalid mining work template")
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, output any) error {
	u := *c.base
	u.Path = path
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		if int64(len(encoded)) > maxResponseBytes {
			return errors.New("mining request exceeds maximum size")
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if int64(len(data)) > maxResponseBytes {
		return errors.New("mining response exceeds maximum size")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mining endpoint returned HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("invalid mining JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("mining response must contain exactly one JSON object")
	}
	return nil
}
