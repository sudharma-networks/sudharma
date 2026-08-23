package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestChainReplaceWithBetterCandidate(t *testing.T) {
	current :=
		NewChain()

	candidate :=
		NewChain()

	currentBlock :=
		buildTestBlock(
			t,
			current.Tip(),
			60,
		)

	if err :=
		current.AddBlock(
			currentBlock,
		); err != nil {

		t.Fatal(err)
	}

	candidateBlock1 :=
		buildTestBlock(
			t,
			candidate.Tip(),
			60,
		)

	if err :=
		candidate.AddBlock(
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

	if err :=
		candidate.AddBlock(
			candidateBlock2,
		); err != nil {

		t.Fatal(err)
	}

	oldCurrentTip :=
		current.Tip().Hash()

	if err :=
		current.ReplaceWith(
			candidate,
		); err != nil {

		t.Fatal(err)
	}

	if current.Height() !=
		candidate.Height() {

		t.Fatalf(
			"expected height %d, got %d",
			candidate.Height(),
			current.Height(),
		)
	}

	if current.Tip().Hash() !=
		candidate.Tip().Hash() {

		t.Fatal(
			"replacement tip does not match candidate",
		)
	}

	if current.Tip().Hash() ==
		oldCurrentTip {

		t.Fatal(
			"chain tip did not change",
		)
	}

	if current.TotalWork().Cmp(
		candidate.TotalWork(),
	) != 0 {

		t.Fatal(
			"replacement total work is wrong",
		)
	}
}

func TestReorganizeToHigherWorkChain(t *testing.T) {
	current :=
		NewChain()

	currentState :=
		NewState()

	candidate :=
		NewChain()

	currentBlock :=
		buildTestBlock(
			t,
			current.Tip(),
			60,
		)

	currentBlock.MinerAddress =
		"current-miner"

	// MinerAddress changes the block hash.
	currentBlock.UpdateMerkleRoot()

	if !mineTestBlock(
		currentBlock,
		1_000_000,
	) {
		t.Fatal(
			"failed to mine current block",
		)
	}

	if err :=
		current.AddBlock(
			currentBlock,
		); err != nil {

		t.Fatal(err)
	}

	if _, err :=
		ProcessBlock(
			currentState,
			currentBlock,
			currentBlock.MinerAddress,
		); err != nil {

		t.Fatal(err)
	}

	candidateBlock1 :=
		buildTestBlock(
			t,
			candidate.Tip(),
			60,
		)

	candidateBlock1.MinerAddress =
		"candidate-miner"

	if !mineTestBlock(
		candidateBlock1,
		1_000_000,
	) {
		t.Fatal(
			"failed to remine candidate block 1",
		)
	}

	if err :=
		candidate.AddBlock(
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

	candidateBlock2.MinerAddress =
		"candidate-miner"

	if !mineTestBlock(
		candidateBlock2,
		1_000_000,
	) {
		t.Fatal(
			"failed to remine candidate block 2",
		)
	}

	if err :=
		candidate.AddBlock(
			candidateBlock2,
		); err != nil {

		t.Fatal(err)
	}

	adopted, err :=
		ReorganizeToCandidate(
			current,
			currentState,
			candidate,
		)

	if err != nil {
		t.Fatal(err)
	}

	if !adopted {
		t.Fatal(
			"better candidate chain was not adopted",
		)
	}

	if current.Height() != 2 {
		t.Fatalf(
			"expected reorganized height 2, got %d",
			current.Height(),
		)
	}

	if current.Tip().Hash() !=
		candidate.Tip().Hash() {

		t.Fatal(
			"reorganized tip does not match candidate",
		)
	}

	expectedSupply :=
		uint64(2) *
			params.InitialBlockReward

	if currentState.IssuedSupply() !=
		expectedSupply {

		t.Fatalf(
			"expected issued supply %d, got %d",
			expectedSupply,
			currentState.IssuedSupply(),
		)
	}

	if currentState.Balance(
		"candidate-miner",
	) != expectedSupply {

		t.Fatalf(
			"candidate miner has wrong balance: expected %d, got %d",
			expectedSupply,
			currentState.Balance("candidate-miner"),
		)
	}

	if currentState.Balance(
		"current-miner",
	) != 0 {

		t.Fatal(
			"old-chain miner balance survived reorganization",
		)
	}
}

func TestReorganizeKeepsBetterCurrentChain(t *testing.T) {
	current :=
		NewChain()

	currentState :=
		NewState()

	candidate :=
		NewChain()

	currentBlock1 :=
		buildTestBlock(
			t,
			current.Tip(),
			60,
		)

	if err :=
		current.AddBlock(
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

	if err :=
		current.AddBlock(
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

	if err :=
		candidate.AddBlock(
			candidateBlock,
		); err != nil {

		t.Fatal(err)
	}

	originalTip :=
		current.Tip().Hash()

	adopted, err :=
		ReorganizeToCandidate(
			current,
			currentState,
			candidate,
		)

	if err != nil {
		t.Fatal(err)
	}

	if adopted {
		t.Fatal(
			"weaker candidate chain was adopted",
		)
	}

	if current.Tip().Hash() !=
		originalTip {

		t.Fatal(
			"current chain changed unexpectedly",
		)
	}
}
