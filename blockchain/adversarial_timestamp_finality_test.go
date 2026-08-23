package blockchain

import (
	"math/big"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestAdversarialSingleTimestampOutlierDoesNotControlNextDifficulty(t *testing.T) {
	chain := NewChain()

	// Fill the history window with target-time blocks so difficulty is stable.
	for i := 0; i < 10; i++ {
		block := buildHistoryTestBlock(t, chain, 60)
		if err := chain.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	before, err := ExpectedNextDifficulty(chain)
	if err != nil {
		t.Fatal(err)
	}

	// One accepted slow timestamp is an outlier inside the 11-interval window.
	// Median history should prevent that single sample from steering difficulty.
	outlier := buildHistoryTestBlock(t, chain, 600)
	if err := chain.AddBlock(outlier); err != nil {
		t.Fatal(err)
	}

	after, err := ExpectedNextDifficulty(chain)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("single timestamp outlier changed next difficulty: before=%d after=%d", before, after)
	}
}

func TestAdversarialFinalizedForkRejectedEvenWithForgedCachedWork(t *testing.T) {
	current := NewChain()

	// Build enough history to move the finalized boundary above genesis.
	for i := uint64(0); i < params.MaxAutomaticReorgDepth+2; i++ {
		block := buildHistoryTestBlock(t, current, 60)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	finalizedHeight, err := FinalizedHeight(current)
	if err != nil {
		t.Fatal(err)
	}
	if finalizedHeight < 1 {
		t.Fatalf("test setup invalid: finalized height %d", finalizedHeight)
	}

	// Fork from genesis so the candidate necessarily conflicts with finalized
	// history if it were ever considered for adoption.
	candidate := NewChain()
	for i := uint64(0); i < 3; i++ {
		block := buildHistoryTestBlock(t, candidate, 10)
		if err := candidate.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	// Simulate an attacker corrupting/forging cached work metadata. The reorg
	// path must rebuild and revalidate candidate work locally before fork choice,
	// so this forged cache must not make the candidate win.
	candidate.mu.Lock()
	candidate.totalWork = new(big.Int).Lsh(big.NewInt(1), 255)
	candidate.mu.Unlock()

	state := NewState()
	adopted, err := ReorganizeToCandidate(current, state, candidate)
	if err != nil {
		t.Fatalf("forged cached work should be neutralized before selection, got error: %v", err)
	}
	if adopted {
		t.Fatal("finalized-history candidate with forged cached work was adopted")
	}
}
