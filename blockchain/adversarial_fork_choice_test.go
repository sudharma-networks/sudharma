package blockchain

import (
	"math/big"
	"testing"
)

func TestAdversarialForgedCachedWorkCannotForceReorg(t *testing.T) {
	current := NewChain()

	for i := 0; i < 2; i++ {
		block := buildHistoryTestBlock(t, current, 60)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	candidate := NewChain()
	candidate.mu.Lock()
	candidate.totalWork = new(big.Int).Lsh(big.NewInt(1), 250)
	candidate.mu.Unlock()

	state := NewState()
	adopted, err := ReorganizeToCandidate(current, state, candidate)
	if err != nil {
		t.Fatalf("unexpected error while rejecting forged work cache: %v", err)
	}
	if adopted {
		t.Fatal("forged cached cumulative work must not force candidate adoption")
	}
	if current.Height() != 2 {
		t.Fatalf("expected current chain height to remain 2, got %d", current.Height())
	}
}

func TestValidateAndCloneChainRecomputesCachedWork(t *testing.T) {
	chain := NewChain()
	block := buildHistoryTestBlock(t, chain, 60)
	if err := chain.AddBlock(block); err != nil {
		t.Fatal(err)
	}

	expectedWork := chain.TotalWork()

	chain.mu.Lock()
	chain.totalWork = new(big.Int).Lsh(big.NewInt(1), 250)
	chain.mu.Unlock()

	validated, err := ValidateAndCloneChain(chain)
	if err != nil {
		t.Fatal(err)
	}
	if validated.TotalWork().Cmp(expectedWork) != 0 {
		t.Fatalf("expected recomputed work %s, got %s", expectedWork, validated.TotalWork())
	}
}
