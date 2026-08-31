package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/params"
)

func TestRunRejectsCPUBackendAndMainnet(t *testing.T) {
	err := run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-backend", "cpu",
		"-probe",
	}, strings.NewReader(""), ioDiscard(), ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "GPU-only") {
		t.Fatalf("cpu backend error = %v", err)
	}

	err = run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-network", "mainnet",
		"-probe",
	}, strings.NewReader(""), ioDiscard(), ioDiscard())
	if err == nil || !strings.Contains(err.Error(), "mainnet") {
		t.Fatalf("mainnet error = %v", err)
	}
}

func TestRunProbeConnectsWithoutCPUFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": params.GPUOnlyMiningMessage})
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-rpc", server.URL,
		"-probe",
	}, strings.NewReader(""), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "GPU-only") && !strings.Contains(out.String(), "Mining work is not live yet") {
		t.Fatalf("output = %q", out.String())
	}
	if strings.Contains(strings.ToLower(out.String()), "cpu mining started") {
		t.Fatal("must not start CPU mining")
	}
}

func TestRunPromptsForAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gpuminer.Work{
			Algorithm:    params.ProductionMiningAlgorithm,
			Version:      2,
			Height:       3,
			Target:       "0f",
			HeaderPrefix: "aa",
		})
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{"-rpc", server.URL, "-probe"}, strings.NewReader("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunOnceMinesWithGPUHasher(t *testing.T) {
	dir := t.TempDir()
	hasher := filepath.Join(dir, "khushi-miner-nvidia")
	script := "#!/bin/sh\necho staging-solution-nonce=7\n"
	if err := os.WriteFile(hasher, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/mining/submit" {
			_ = json.NewEncoder(w).Encode(gpuminer.SubmitResult{Status: "accepted"})
			return
		}
		_ = json.NewEncoder(w).Encode(gpuminer.Work{
			WorkID:        "work-1",
			Algorithm:     params.ProductionMiningAlgorithm,
			Version:       2,
			Height:        4,
			Target:        "0f",
			HeaderPrefix:  "aa",
			RewardAddress: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		})
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{
		"-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-rpc", server.URL,
		"-hasher-dir", dir,
		"-once",
	}, strings.NewReader(""), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "accepted GPU share") {
		t.Fatalf("output = %q", out.String())
	}
	if strings.Contains(strings.ToLower(out.String()), "cpu mining") {
		t.Fatal("must not mention CPU mining")
	}
}

func TestRunOnceMinesCandidateBlockWithoutHasher(t *testing.T) {
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	previous := mustGenesisCandidate(t, address)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/mining/submit" {
			_ = json.NewEncoder(w).Encode(gpuminer.SubmitResult{Status: "accepted", Accepted: true, RewardAddress: address, Balance: 1})
			return
		}
		_ = json.NewEncoder(w).Encode(gpuminer.Work{
			Algorithm:     params.ProductionMiningAlgorithm,
			Height:        previous.Height,
			Difficulty:    previous.Difficulty,
			RewardAddress: address,
			Block:         previous,
		})
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{
		"-address", address,
		"-rpc", server.URL,
		"-hasher-dir", t.TempDir(),
		"-once",
	}, strings.NewReader(""), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "accepted GPU share") {
		t.Fatalf("output = %q", out.String())
	}
	if strings.Contains(out.String(), "demand miner") && strings.Contains(strings.ToLower(out.String()), "starting demand") {
		t.Fatal("must not start the demand miner")
	}
}

func TestRunAutoUsesSavedAddress(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUDHARMA_MINER_DATA_DIR", dir)
	address := "cccccccccccccccccccccccccccccccccccccccc"
	if err := gpuminer.SaveAddress(address); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"network": "sudharma", "height": 12})
		default:
			_ = json.NewEncoder(w).Encode(gpuminer.Work{
				Algorithm:    params.ProductionMiningAlgorithm,
				Height:       3,
				Target:       "0f",
				HeaderPrefix: "aa",
			})
		}
	}))
	t.Cleanup(server.Close)

	out := &strings.Builder{}
	err := run([]string{"-rpc", server.URL, "-auto", "-probe"}, strings.NewReader(""), out, ioDiscard())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Using saved wallet address") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), address[:6]) {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDetectGPUHasher(t *testing.T) {
	dir := t.TempDir()
	if _, err := gpuminer.DetectGPUHasher(dir); err == nil {
		t.Fatal("expected missing hasher")
	}
	path := filepath.Join(dir, "khushi-miner-nvidia")
	if err := os.WriteFile(path, []byte("gpu"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := gpuminer.DetectGPUHasher(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("hasher = %q", got)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func ioDiscard() *discard { return &discard{} }

func mustGenesisCandidate(t *testing.T, minerAddr string) *blockchain.Block {
	t.Helper()
	previous := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(previous, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	if block.Timestamp <= previous.Timestamp {
		block.Timestamp = previous.Timestamp + 1
	}
	block.Difficulty = 1
	block.MinerAddress = minerAddr
	block.Nonce = 0
	block.UpdateMerkleRoot()
	return block
}
