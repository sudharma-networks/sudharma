package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func buildFinalityTestChains(t *testing.T) (*Chain, *Chain, uint64) {
	t.Helper()

	current := NewChain()
	candidate := NewChain()

	// Share enough early history that we can fork immediately before the
	// finalized boundary once the current chain is extended.
	for i := 0; i < 5; i++ {
		block := buildHistoryTestBlock(t, current, 60)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
		if err := candidate.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	for i := uint64(0); i < params.MaxAutomaticReorgDepth; i++ {
		block := buildHistoryTestBlock(t, current, 60)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	finalizedHeight, err := FinalizedHeight(current)
	if err != nil {
		t.Fatal(err)
	}
	return current, candidate, finalizedHeight
}

func TestValidateFinalizedReorgRejectsReplacingFinalizedBlock(t *testing.T) {
	current, candidate, finalizedHeight := buildFinalityTestChains(t)
	if finalizedHeight != 5 {
		t.Fatalf("expected finalized height 5, got %d", finalizedHeight)
	}

	// Rebuild a candidate that shares only heights 0..4, so block 5 would be
	// replaced even though the current node considers it finalized.
	fork := NewChain()
	for height := uint64(1); height < finalizedHeight; height++ {
		block, ok := candidate.BlockByHeight(height)
		if !ok {
			t.Fatalf("missing shared block %d", height)
		}
		if err := fork.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}
	alternate := buildHistoryTestBlock(t, fork, 30)
	if err := fork.AddBlock(alternate); err != nil {
		t.Fatal(err)
	}

	if err := ValidateFinalizedReorg(current, fork); err == nil {
		t.Fatal("expected replacement of finalized history to be rejected")
	}
}

func TestValidateFinalizedReorgAllowsForkAfterFinalizedHeight(t *testing.T) {
	current, candidate, finalizedHeight := buildFinalityTestChains(t)
	if finalizedHeight != 5 {
		t.Fatalf("expected finalized height 5, got %d", finalizedHeight)
	}

	// candidate already shares through height 5. A fork beginning at height 6
	// leaves all finalized history untouched and therefore passes this guard.
	alternate := buildHistoryTestBlock(t, candidate, 30)
	if err := candidate.AddBlock(alternate); err != nil {
		t.Fatal(err)
	}

	if err := ValidateFinalizedReorg(current, candidate); err != nil {
		t.Fatalf("expected fork after finalized height to be allowed: %v", err)
	}
}
