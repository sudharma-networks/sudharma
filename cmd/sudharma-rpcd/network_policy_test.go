package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func TestRPCDLoadOrCreateChainUsesExplicitNetwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mainnet-chain.json")
	data, err := json.Marshal([]*blockchain.Block{blockchain.NewMainnetGenesisBlock()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	chain, err := loadOrCreateChain(path, params.NetworkMainnet)
	if err != nil {
		t.Fatalf("load mainnet candidate: %v", err)
	}
	if got := chain.Network(); got != params.NetworkMainnet {
		t.Fatalf("network = %q, want %q", got, params.NetworkMainnet)
	}
	if _, err := loadOrCreateChain(path, params.NetworkPublicTestnet); err == nil {
		t.Fatal("mainnet chain file accepted as public testnet")
	}
}

func TestRPCDLoadOrCreateStateUsesExplicitPolicy(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "mainnet-chain.json")
	data, err := json.Marshal([]*blockchain.Block{blockchain.NewMainnetGenesisBlock()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chainPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	chain, err := blockchain.LoadChainFromFileFor(chainPath, params.NetworkMainnet)
	if err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(t.TempDir(), "mainnet-state.json")
	state, err := loadOrCreateState(chain, statePath, params.MonetaryPolicyMainnet)
	if err != nil {
		t.Fatalf("create mainnet state: %v", err)
	}
	if got := state.MonetaryPolicy(); got != params.MonetaryPolicyMainnet {
		t.Fatalf("state policy = %d, want %d", got, params.MonetaryPolicyMainnet)
	}
}
