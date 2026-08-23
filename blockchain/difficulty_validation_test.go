package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
)

func TestWrongDifficultyRejected(t *testing.T) {
	previous := NewGenesisBlock()

	blockTime :=
		previous.Timestamp + 60

	expectedDifficulty :=
		consensus.NextDifficulty(
			previous.Difficulty,
			60,
		)

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    blockTime,
		PreviousHash: previous.Hash(),

		// Deliberately wrong.
		Difficulty: expectedDifficulty + 1,

		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if err := ValidateBlockBasic(
		block,
		previous,
	); err == nil {
		t.Fatal(
			"block with incorrect difficulty was accepted",
		)
	}
}

func TestFastBlockRequiresHigherDifficulty(t *testing.T) {
	previous := NewGenesisBlock()

	// Give the previous block a meaningful
	// development difficulty.
	previous.Difficulty = 100

	actualBlockTime := int64(20)

	expectedDifficulty :=
		consensus.NextDifficulty(
			previous.Difficulty,
			actualBlockTime,
		)

	if expectedDifficulty <= previous.Difficulty {
		t.Fatal(
			"expected difficulty to increase",
		)
	}

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + actualBlockTime,
		PreviousHash: previous.Hash(),

		// Miner incorrectly keeps old difficulty.
		Difficulty: previous.Difficulty,

		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if err := ValidateBlockBasic(
		block,
		previous,
	); err == nil {
		t.Fatal(
			"fast block using old difficulty was accepted",
		)
	}
}

func TestSlowBlockRequiresLowerDifficulty(t *testing.T) {
	previous := NewGenesisBlock()

	previous.Difficulty = 100

	actualBlockTime := int64(180)

	expectedDifficulty :=
		consensus.NextDifficulty(
			previous.Difficulty,
			actualBlockTime,
		)

	if expectedDifficulty >= previous.Difficulty {
		t.Fatal(
			"expected difficulty to decrease",
		)
	}

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + actualBlockTime,
		PreviousHash: previous.Hash(),

		// Miner incorrectly keeps old difficulty.
		Difficulty: previous.Difficulty,

		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if err := ValidateBlockBasic(
		block,
		previous,
	); err == nil {
		t.Fatal(
			"slow block using old difficulty was accepted",
		)
	}
}
