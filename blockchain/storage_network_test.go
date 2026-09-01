package blockchain

import (
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestLoadChainFromFileForPreservesPublicTestnetIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.json")
	chain := NewChain()
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadChainFromFileFor(path, params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Network(); got != params.NetworkPublicTestnet {
		t.Fatalf("loaded network = %q, want %q", got, params.NetworkPublicTestnet)
	}
}

func TestLoadChainFromFileForAcceptsOfflineMainnetGenesis(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mainnet-chain.json")
	chain := offlineMainnetChainForTest(t)
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadChainFromFileFor(path, params.NetworkMainnet)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Network(); got != params.NetworkMainnet {
		t.Fatalf("loaded network = %q, want %q", got, params.NetworkMainnet)
	}
}

func TestLoadChainFromFileForRejectsWrongNetwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.json")
	chain := NewChain()
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadChainFromFileFor(path, params.NetworkMainnet); err == nil {
		t.Fatal("expected testnet chain loaded as mainnet to fail closed")
	}
}
