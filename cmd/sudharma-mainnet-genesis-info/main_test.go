package main

import (
	"encoding/json"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func TestMainnetGenesisInfoReportsCandidateWithoutAuthorizingLaunch(t *testing.T) {
	payload := buildGenesisInfo()
	if payload.Network != string(params.NetworkMainnet) {
		t.Fatalf("network = %q", payload.Network)
	}
	if payload.LaunchAuthorized || payload.MiningAuthorized {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Hash != blockchain.NewMainnetGenesisBlock().Hash() {
		t.Fatalf("hash = %q", payload.Hash)
	}
	if payload.MerkleRoot != "Sudharma Network Mainnet Genesis Block v1" {
		t.Fatalf("merkle root = %q", payload.MerkleRoot)
	}
	if payload.Algorithm != params.ProductionMiningAlgorithm {
		t.Fatalf("algorithm = %q", payload.Algorithm)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("expected JSON output")
	}
}
