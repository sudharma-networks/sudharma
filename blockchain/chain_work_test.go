package blockchain

import (
	"math/big"
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
)

func TestGenesisHasWork(t *testing.T) {
	chain := NewChain()

	work := chain.TotalWork()

	if work.Sign() <= 0 {
		t.Fatal(
			"genesis chain work must be positive",
		)
	}
}

func TestChainWorkIncreases(t *testing.T) {
	chain := NewChain()

	before :=
		chain.TotalWork()

	previous :=
		chain.Tip()

	blockTime :=
		int64(60)

	difficulty :=
		consensus.NextDifficulty(
			previous.Difficulty,
			blockTime,
		)

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + blockTime,
		PreviousHash: previous.Hash(),
		Difficulty:   difficulty,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if !mineTestBlock(
		block,
		1_000_000,
	) {
		t.Fatal(
			"failed to mine test block",
		)
	}

	if err := chain.AddBlock(
		block,
	); err != nil {
		t.Fatal(err)
	}

	after :=
		chain.TotalWork()

	if after.Cmp(before) <= 0 {
		t.Fatal(
			"chain work did not increase",
		)
	}

	expected :=
		new(big.Int).Add(
			before,
			blockWork(block.Difficulty),
		)

	if after.Cmp(expected) != 0 {
		t.Fatalf(
			"unexpected total work: expected %s, got %s",
			expected.String(),
			after.String(),
		)
	}
}

func TestHigherDifficultyAddsMoreWork(t *testing.T) {
	low :=
		blockWork(10)

	high :=
		blockWork(100)

	if high.Cmp(low) <= 0 {
		t.Fatal(
			"higher difficulty should represent more work",
		)
	}
}
