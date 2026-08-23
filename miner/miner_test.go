package miner

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/pow"
)

func TestMine(t *testing.T) {
	block := blockchain.NewGenesisBlock()

	// Very easy development difficulty.
	block.Difficulty = 1

	result := Mine(block, 0, 100000)

	if !result.Found {
		t.Fatal("miner failed to find a valid block")
	}

	if !pow.ValidHash(result.Hash, block.Difficulty) {
		t.Fatal("miner returned an invalid PoW hash")
	}
}
