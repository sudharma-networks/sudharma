package gpuminer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/params"
)

func TestResolvePublicTestnetAndRejectsMainnet(t *testing.T) {
	cfg, err := Resolve(Config{
		Address: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Network: "public-testnet",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RPCURL != params.PublicTestnetMiningRPC {
		t.Fatalf("rpc = %q", cfg.RPCURL)
	}
	if cfg.Backend != params.ProductionMiningBackend {
		t.Fatalf("backend = %q", cfg.Backend)
	}

	_, err = Resolve(Config{
		Address: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Network: "mainnet",
	})
	if err == nil {
		t.Fatal("mainnet must be refused")
	}
}

func TestResolveRejectsCPUAndASICBackends(t *testing.T) {
	for _, backend := range []string{"cpu", "asic"} {
		_, err := Resolve(Config{
			Address: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Network: "public-testnet",
			Backend: backend,
		})
		if err == nil || !strings.Contains(err.Error(), "GPU-only") {
			t.Fatalf("backend %q error = %v", backend, err)
		}
	}
}

func TestWorkFromJSONRejectsCPUAlgorithm(t *testing.T) {
	_, err := WorkFromJSON([]byte(`{"algorithm":"sudharma-cpu-v1","target":"aa","header_prefix":"bb"}`))
	if err == nil || !strings.Contains(err.Error(), "GPU-only") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoopAcceptsGPUShareAndRejectsCPUBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/mining/work":
			_ = json.NewEncoder(w).Encode(Work{
				WorkID:        "work-1",
				Algorithm:     params.ProductionMiningAlgorithm,
				Version:       2,
				Height:        12,
				Difficulty:    1,
				Target:        "0f",
				HeaderPrefix:  "aa",
				RewardAddress: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			})
		case r.URL.Path == "/v1/mining/submit":
			_ = json.NewEncoder(w).Encode(SubmitResult{Status: "accepted"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	loop := &Loop{
		Client:  client,
		Address: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Backend: StaticNonceBackend{BackendName: "cuda", Nonce: 7},
		Sleep:   func(time.Duration) {},
		Log:     func(string, ...any) {},
	}
	accepted, err := loop.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if accepted < 1 {
		t.Fatalf("accepted = %d", accepted)
	}

	cpuLoop := &Loop{Client: client, Address: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Backend: RejectedBackend{BackendName: "cpu"}}
	if _, err := cpuLoop.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "GPU-only") {
		t.Fatalf("cpu backend error = %v", err)
	}
}

func TestValidateRewardAddress(t *testing.T) {
	if err := ValidateRewardAddress("not-an-address"); err == nil {
		t.Fatal("expected invalid address")
	}
	if err := ValidateRewardAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err != nil {
		t.Fatal(err)
	}
}

func TestWorkFromJSONAcceptsCandidateBlockWithoutKhushiHeader(t *testing.T) {
	block := candidateTestBlock(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	raw, err := json.Marshal(Work{
		Algorithm:     params.ProductionMiningAlgorithm,
		Height:        block.Height,
		Difficulty:    block.Difficulty,
		RewardAddress: block.MinerAddress,
		Block:         block,
	})
	if err != nil {
		t.Fatal(err)
	}
	work, err := WorkFromJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if work.Block == nil || work.Block.MinerAddress != block.MinerAddress {
		t.Fatalf("block = %+v", work.Block)
	}
}

func TestLoopMinesCandidateBlockWithoutGPUHasher(t *testing.T) {
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	block := candidateTestBlock(t, address)
	var submitted *blockchain.Block
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/mining/work":
			_ = json.NewEncoder(w).Encode(Work{
				Algorithm:     params.ProductionMiningAlgorithm,
				Height:        block.Height,
				Difficulty:    block.Difficulty,
				RewardAddress: address,
				Block:         block,
			})
		case r.URL.Path == "/v1/mining/submit":
			var got blockchain.Block
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			submitted = &got
			_ = json.NewEncoder(w).Encode(SubmitResult{Status: "accepted", Accepted: true, Balance: 50, RewardAddress: address})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	loop := &Loop{
		Client:  client,
		Address: address,
		Once:    true,
		Sleep:   func(time.Duration) {},
		Log:     func(string, ...any) {},
	}
	accepted, err := loop.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if accepted != 1 {
		t.Fatalf("accepted = %d", accepted)
	}
	if submitted == nil || submitted.MinerAddress != address || submitted.Nonce == 0 && submitted.Hash() == "" {
		t.Fatalf("submitted = %+v", submitted)
	}
	if submitted.MinerAddress != address {
		t.Fatalf("miner address = %q", submitted.MinerAddress)
	}
}

func candidateTestBlock(t *testing.T, minerAddr string) *blockchain.Block {
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
