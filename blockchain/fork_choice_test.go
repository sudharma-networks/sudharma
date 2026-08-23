package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
)

func buildTestBlock(
	t *testing.T,
	previous *Block,
	blockTime int64,
) *Block {

	t.Helper()

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

	return block
}

func TestCandidateWithMoreWorkWins(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	// Add one normal block to current chain.
	currentBlock :=
		buildTestBlock(
			t,
			current.Tip(),
			60,
		)

	if err := current.AddBlock(
		currentBlock,
	); err != nil {
		t.Fatal(err)
	}

	// Candidate gets two valid blocks.
	candidateBlock1 :=
		buildTestBlock(
			t,
			candidate.Tip(),
			60,
		)

	if err := candidate.AddBlock(
		candidateBlock1,
	); err != nil {
		t.Fatal(err)
	}

	candidateBlock2 :=
		buildTestBlock(
			t,
			candidate.Tip(),
			60,
		)

	if err := candidate.AddBlock(
		candidateBlock2,
	); err != nil {
		t.Fatal(err)
	}

	best, err :=
		BetterChain(
			current,
			candidate,
		)

	if err != nil {
		t.Fatal(err)
	}

	if best != candidate {
		t.Fatal(
			"candidate with more work was not selected",
		)
	}
}

func TestCurrentChainKeptWhenItHasMoreWork(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	currentBlock1 :=
		buildTestBlock(
			t,
			current.Tip(),
			60,
		)

	if err := current.AddBlock(
		currentBlock1,
	); err != nil {
		t.Fatal(err)
	}

	currentBlock2 :=
		buildTestBlock(
			t,
			current.Tip(),
			60,
		)

	if err := current.AddBlock(
		currentBlock2,
	); err != nil {
		t.Fatal(err)
	}

	candidateBlock :=
		buildTestBlock(
			t,
			candidate.Tip(),
			60,
		)

	if err := candidate.AddBlock(
		candidateBlock,
	); err != nil {
		t.Fatal(err)
	}

	best, err :=
		BetterChain(
			current,
			candidate,
		)

	if err != nil {
		t.Fatal(err)
	}

	if best != current {
		t.Fatal(
			"current chain with more work was not kept",
		)
	}
}

func TestEqualChainsKeepCurrent(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	best, err :=
		BetterChain(
			current,
			candidate,
		)

	if err != nil {
		t.Fatal(err)
	}

	if best != current {
		t.Fatal(
			"equal candidate should not replace current chain",
		)
	}
}

func TestNilCandidateRejected(t *testing.T) {
	current := NewChain()

	_, err :=
		BetterChain(
			current,
			nil,
		)

	if err == nil {
		t.Fatal(
			"nil candidate chain was accepted",
		)
	}
}
