package blockchain

import "testing"

func TestValidateFinalizedReorgRejectsNilInputs(t *testing.T) {
	if err := ValidateFinalizedReorg(nil, NewChain()); err == nil {
		t.Fatal("expected nil current chain to be rejected")
	}
	if err := ValidateFinalizedReorg(NewChain(), nil); err == nil {
		t.Fatal("expected nil candidate chain to be rejected")
	}
}

func TestFinalityGenesisAndTipConsistency(t *testing.T) {
	chain := NewChain()

	confirmations, err := Confirmations(chain, 0)
	if err != nil {
		t.Fatal(err)
	}
	if confirmations != 1 {
		t.Fatalf("expected genesis to have 1 confirmation on genesis-only chain, got %d", confirmations)
	}

	finalized, err := IsBlockFinalized(chain, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !finalized {
		t.Fatal("expected genesis to be finalized")
	}
}

func TestValidateFinalizedReorgAllowsIdenticalChain(t *testing.T) {
	chain := NewChain()
	if err := ValidateFinalizedReorg(chain, chain); err != nil {
		t.Fatalf("expected identical chain to satisfy finalized-history guard: %v", err)
	}
}
