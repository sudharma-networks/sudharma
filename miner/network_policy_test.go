package miner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/params"
)

func TestMineNextBlockUsesChainMonetaryPolicy(t *testing.T) {
	genesis := blockchain.NewMainnetGenesisBlock()
	data, err := json.Marshal([]*blockchain.Block{genesis})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mainnet-chain.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	chain, err := blockchain.LoadChainFromFileFor(path, params.NetworkMainnet)
	if err != nil {
		t.Fatal(err)
	}
	state := blockchain.NewStateFor(params.MonetaryPolicyMainnet)
	pool := mempool.NewMempool()
	minerAddress := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	_, reward, err := MineNextBlock(chain, state, pool, minerAddress, 1_000_000)
	if err != nil {
		t.Fatalf("mainnet-policy mining failed: %v", err)
	}
	if reward == 0 {
		t.Fatal("expected a non-zero mainnet block reward at height 1")
	}
	if state.MonetaryPolicy() != params.MonetaryPolicyMainnet {
		t.Fatalf("state policy changed: got %d", state.MonetaryPolicy())
	}
	if chain.Network() != params.NetworkMainnet {
		t.Fatalf("chain network changed: got %q", chain.Network())
	}
}
