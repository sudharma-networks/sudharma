package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
)

func TestNewChainStartsWithGenesis(t *testing.T) {
	chain := NewChain()

	if chain.Length() != 1 {
		t.Fatalf(
			"expected chain length 1, got %d",
			chain.Length(),
		)
	}

	if chain.Height() != 0 {
		t.Fatalf(
			"expected height 0, got %d",
			chain.Height(),
		)
	}

	tip := chain.Tip()

	if tip == nil {
		t.Fatal("chain tip should not be nil")
	}

	if tip.Height != 0 {
		t.Fatalf(
			"expected genesis height 0, got %d",
			tip.Height,
		)
	}
}

func TestChainAcceptsValidBlock(t *testing.T) {
	chain := NewChain()

	previous := chain.Tip()

	actualBlockTime := int64(60)

	difficulty := consensus.NextDifficulty(
		previous.Difficulty,
		actualBlockTime,
	)

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + actualBlockTime,
		PreviousHash: previous.Hash(),
		Difficulty:   difficulty,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if !mineTestBlock(block, 1_000_000) {
		t.Fatal("failed to mine test block")
	}

	if err := chain.AddBlock(block); err != nil {
		t.Fatalf(
			"valid block was rejected: %v",
			err,
		)
	}

	if chain.Height() != 1 {
		t.Fatalf(
			"expected height 1, got %d",
			chain.Height(),
		)
	}

	if chain.Length() != 2 {
		t.Fatalf(
			"expected chain length 2, got %d",
			chain.Length(),
		)
	}
}

func TestChainRejectsWrongPreviousHash(t *testing.T) {
	chain := NewChain()

	previous := chain.Tip()

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + 60,
		PreviousHash: "not-the-real-previous-hash",
		Difficulty: consensus.NextDifficulty(
			previous.Difficulty,
			60,
		),
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if err := chain.AddBlock(block); err == nil {
		t.Fatal(
			"block with wrong previous hash was accepted",
		)
	}

	if chain.Height() != 0 {
		t.Fatal(
			"chain height changed after rejected block",
		)
	}
}

func TestBlockByHeight(t *testing.T) {
	chain := NewChain()

	block, ok := chain.BlockByHeight(0)

	if !ok {
		t.Fatal("genesis block was not found")
	}

	if block.Height != 0 {
		t.Fatalf(
			"expected block height 0, got %d",
			block.Height,
		)
	}

	_, ok = chain.BlockByHeight(100)

	if ok {
		t.Fatal(
			"nonexistent block height was returned",
		)
	}
}
