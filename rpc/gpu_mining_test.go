package rpc

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
)

func TestMiningEndpointsRejectCPUAlgorithmAndWrongMethods(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	submit := request(t, server, http.MethodPost, "/v1/mining/submit", []byte(`{"algorithm":"sudharma-cpu-v1"}`), "application/json")
	if submit.Code != http.StatusServiceUnavailable {
		t.Fatalf("submit status = %d", submit.Code)
	}
	if !strings.Contains(submit.Body.String(), params.GPUOnlyMiningMessage) {
		t.Fatalf("submit body = %q", submit.Body.String())
	}

	put := request(t, server, http.MethodPut, "/v1/mining/work", nil, "")
	if put.Code != http.StatusMethodNotAllowed {
		t.Fatalf("put work status = %d", put.Code)
	}
	getSubmit := request(t, server, http.MethodGet, "/v1/mining/submit", nil, "")
	if getSubmit.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get submit status = %d", getSubmit.Code)
	}
}

func TestGPUMinerWorkDoesNotAdvertiseCPUMining(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	work := request(t, server, http.MethodPost, "/v1/mining/work", []byte(`{"address":"`+address+`"}`), "application/json")
	if work.Code != http.StatusOK {
		t.Fatalf("work status = %d body = %s", work.Code, work.Body.String())
	}
	if strings.Contains(work.Body.String(), "sudharma-cpu-v1") {
		t.Fatal("must not advertise a CPU mining algorithm")
	}

	var payload struct {
		Algorithm     string            `json:"algorithm"`
		Backend       string            `json:"backend"`
		RewardAddress string            `json:"reward_address"`
		Height        uint64            `json:"height"`
		Block         *blockchain.Block `json:"block"`
		POWCompat     struct {
			GetBlockTemplate map[string]any `json:"getblocktemplate"`
			EthGetWork       map[string]any `json:"eth_getWork"`
		} `json:"pow_compat"`
		Note string `json:"note"`
	}
	if err := json.Unmarshal(work.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Algorithm != params.ProductionMiningAlgorithm {
		t.Fatalf("algorithm = %q", payload.Algorithm)
	}
	if payload.Backend != params.ProductionMiningBackend {
		t.Fatalf("backend = %q", payload.Backend)
	}
	if payload.RewardAddress != address {
		t.Fatalf("reward = %q", payload.RewardAddress)
	}
	if payload.Block == nil || payload.Block.MinerAddress != address {
		t.Fatalf("candidate miner address = %+v", payload.Block)
	}
	if payload.Height != 1 {
		t.Fatalf("height = %d", payload.Height)
	}
	if payload.POWCompat.GetBlockTemplate["height"] != float64(1) && payload.POWCompat.GetBlockTemplate["height"] != uint64(1) {
		t.Fatalf("pow_compat.getblocktemplate.height = %#v", payload.POWCompat.GetBlockTemplate["height"])
	}
	if payload.POWCompat.EthGetWork["header_hash"] == nil || payload.POWCompat.EthGetWork["header_hash"] == "" {
		t.Fatalf("pow_compat.eth_getWork.header_hash = %#v", payload.POWCompat.EthGetWork["header_hash"])
	}
	if !strings.Contains(payload.Note, "Demand miner") {
		t.Fatalf("note = %q", payload.Note)
	}

	getWork := request(t, server, http.MethodGet, "/v1/mining/work?address="+address, nil, "")
	if getWork.Code != http.StatusOK {
		t.Fatalf("get work status = %d", getWork.Code)
	}

	missing := request(t, server, http.MethodPost, "/v1/mining/work", []byte(`{}`), "application/json")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing address status = %d", missing.Code)
	}
}

func TestGPUMinerSubmitCreditsRewardAddress(t *testing.T) {
	server, _, chain, state := newTestServer(t)
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	work := request(t, server, http.MethodPost, "/v1/mining/work", []byte(`{"address":"`+address+`"}`), "application/json")
	if work.Code != http.StatusOK {
		t.Fatalf("work status = %d body = %s", work.Code, work.Body.String())
	}
	var payload struct {
		Block *blockchain.Block `json:"block"`
	}
	if err := json.Unmarshal(work.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Block == nil {
		t.Fatal("missing candidate block")
	}

	result := miner.Mine(payload.Block, 0, 1_000_000)
	if !result.Found {
		t.Fatal("failed to mine GPU miner candidate")
	}

	raw, err := json.Marshal(payload.Block)
	if err != nil {
		t.Fatal(err)
	}
	submit := request(t, server, http.MethodPost, "/v1/mining/submit", raw, "application/json")
	if submit.Code != http.StatusOK {
		t.Fatalf("submit status = %d body = %s", submit.Code, submit.Body.String())
	}

	var accepted struct {
		Status        string `json:"status"`
		Accepted      bool   `json:"accepted"`
		RewardAddress string `json:"reward_address"`
		Balance       uint64 `json:"balance"`
		Height        uint64 `json:"height"`
	}
	if err := json.Unmarshal(submit.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "accepted" || !accepted.Accepted {
		t.Fatalf("accepted = %+v", accepted)
	}
	if accepted.RewardAddress != address {
		t.Fatalf("reward address = %q", accepted.RewardAddress)
	}
	want := consensus.BlockSubsidy(1)
	if accepted.Balance != want || state.Balance(address) != want {
		t.Fatalf("balance = %d want %d", accepted.Balance, want)
	}
	if chain.Height() != 1 {
		t.Fatalf("height = %d", chain.Height())
	}
}

func TestMiningEndpointsRejectWrongMethods(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	put := request(t, server, http.MethodPut, "/v1/mining/work", nil, "")
	if put.Code != http.StatusMethodNotAllowed {
		t.Fatalf("put work status = %d", put.Code)
	}
	getSubmit := request(t, server, http.MethodGet, "/v1/mining/submit", nil, "")
	if getSubmit.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get submit status = %d", getSubmit.Code)
	}
}
