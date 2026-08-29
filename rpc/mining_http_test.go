package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func testMiningHTTPBlock() *blockchain.Block {
	return &blockchain.Block{
		Version:      2,
		Height:       9000,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   1,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
}

func TestMiningHTTPAPIGetWorkAndSubmitSolution(t *testing.T) {
	service := NewMiningWorkService(func(block *blockchain.Block, nonce uint64) bool {
		return block != nil && nonce == 42
	})
	api := NewMiningHTTPAPI(service, func() (*blockchain.Block, error) {
		return testMiningHTTPBlock(), nil
	}, nil)

	getReq := httptest.NewRequest(http.MethodGet, "/v1/mining/work", nil)
	getRec := httptest.NewRecorder()
	api.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get-work status: got %d body=%s", getRec.Code, getRec.Body.String())
	}

	var work MiningWorkTemplate
	if err := json.NewDecoder(getRec.Body).Decode(&work); err != nil {
		t.Fatalf("decode work template: %v", err)
	}
	if work.WorkID == "" || work.Algorithm == "" || work.RewardAddress == "" {
		t.Fatalf("incomplete work template: %+v", work)
	}

	solution := MiningSolution{
		WorkID:          work.WorkID,
		Nonce:           42,
		Algorithm:       work.Algorithm,
		Version:         work.Version,
		Height:          work.Height,
		Difficulty:      work.Difficulty,
		TargetHex:       work.TargetHex,
		HeaderPrefixHex: work.HeaderPrefixHex,
		RewardAddress:   work.RewardAddress,
	}
	body, err := json.Marshal(solution)
	if err != nil {
		t.Fatal(err)
	}
	postReq := httptest.NewRequest(http.MethodPost, "/v1/mining/submit", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	api.Handler().ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("submit-work status: got %d body=%s", postRec.Code, postRec.Body.String())
	}

	var result MiningSubmitResult
	if err := json.NewDecoder(postRec.Body).Decode(&result); err != nil {
		t.Fatalf("decode submit result: %v", err)
	}
	if result.Status != MiningSubmitAccepted {
		t.Fatalf("submit status: got %q want %q", result.Status, MiningSubmitAccepted)
	}

	telemetry := api.Telemetry()
	if telemetry.WorkIssued != 1 || telemetry.SubmitAccepted != 1 {
		t.Fatalf("unexpected telemetry: %+v", telemetry)
	}
}

func TestMiningHTTPAPIIsConstrainedAndRateLimitReady(t *testing.T) {
	service := NewMiningWorkService(func(*blockchain.Block, uint64) bool { return true })
	api := NewMiningHTTPAPI(service, func() (*blockchain.Block, error) {
		return testMiningHTTPBlock(), nil
	}, func(r *http.Request) bool {
		return r.Header.Get("X-Allow-Mining") == "yes"
	})

	blocked := httptest.NewRecorder()
	api.Handler().ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "/v1/mining/work", nil))
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limit boundary: got %d want %d", blocked.Code, http.StatusTooManyRequests)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	adminReq.Header.Set("X-Allow-Mining", "yes")
	adminRec := httptest.NewRecorder()
	api.Handler().ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusNotFound {
		t.Fatalf("mining handler exposed non-mining route: got %d", adminRec.Code)
	}

	wrongMethod := httptest.NewRequest(http.MethodPost, "/v1/mining/work", nil)
	wrongMethod.Header.Set("X-Allow-Mining", "yes")
	wrongMethodRec := httptest.NewRecorder()
	api.Handler().ServeHTTP(wrongMethodRec, wrongMethod)
	if wrongMethodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get-work wrong method: got %d", wrongMethodRec.Code)
	}

	telemetry := api.Telemetry()
	if telemetry.RateLimited != 1 {
		t.Fatalf("rate-limited counter: got %d want 1", telemetry.RateLimited)
	}
}
