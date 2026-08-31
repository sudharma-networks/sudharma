package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestPublicTestnetGenesisIdentityUnchanged(t *testing.T) {
	genesis := NewGenesisBlock()
	if genesis.Timestamp != 1786924800 {
		t.Fatalf("testnet genesis timestamp changed: %d", genesis.Timestamp)
	}
	if genesis.MerkleRoot != "Sudharma Network Genesis Block" {
		t.Fatalf("testnet genesis merkle changed: %q", genesis.MerkleRoot)
	}
	got, err := GenesisFor(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash() != genesis.Hash() {
		t.Fatal("GenesisFor(testnet) diverged from NewGenesisBlock")
	}
}

func TestMainnetGenesisCandidateIsIsolated(t *testing.T) {
	testnet := NewGenesisBlock()
	mainnet := NewMainnetGenesisBlock()
	if mainnet.Hash() == testnet.Hash() {
		t.Fatal("mainnet genesis hash collides with public testnet genesis")
	}
	if mainnet.MerkleRoot != "Sudharma Network Mainnet Genesis Block v1" {
		t.Fatalf("mainnet genesis merkle changed: %q", mainnet.MerkleRoot)
	}
	if mainnet.Timestamp != int64(params.MainnetGenesisTimestamp) {
		t.Fatalf("mainnet genesis timestamp changed: %d", mainnet.Timestamp)
	}
	if mainnet.Hash() != NewMainnetGenesisBlock().Hash() {
		t.Fatal("mainnet genesis hash is not deterministic")
	}
}

func TestGenesisForRejectsUnauthorizedMainnet(t *testing.T) {
	if params.MainnetLaunchAuthorized {
		t.Fatal("mainnet launch became authorized")
	}
	if _, err := GenesisFor(params.NetworkMainnet); err == nil {
		t.Fatal("expected unauthorized mainnet genesis to be rejected")
	}
}

func TestNewChainStillUsesPublicTestnetGenesis(t *testing.T) {
	chain := NewChain()
	genesis, ok := chain.BlockByHeight(0)
	if !ok {
		t.Fatal("missing genesis")
	}
	if genesis.Hash() != NewGenesisBlock().Hash() {
		t.Fatal("default chain is not the public-testnet genesis")
	}
}

func TestNewChainForRejectsUnauthorizedMainnet(t *testing.T) {
	if params.MainnetLaunchAuthorized {
		t.Fatal("mainnet launch became authorized")
	}
	if _, err := NewChainFor(params.NetworkMainnet); err == nil {
		t.Fatal("expected unauthorized mainnet chain creation to fail")
	}
}

func TestValidateChainGenesisMatchesNetwork(t *testing.T) {
	chain := NewChain()
	if err := ValidateChainGenesis(chain, params.NetworkPublicTestnet); err != nil {
		t.Fatal(err)
	}
	if err := ValidateChainGenesis(chain, params.NetworkMainnet); err == nil {
		t.Fatal("expected mainnet validation to fail for testnet genesis")
	}
}
