package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestFinalizedHeightBeforeReorgWindowIsGenesis(t *testing.T) {
	chain := NewChain()

	finalized, err := FinalizedHeight(chain)
	if err != nil {
		t.Fatal(err)
	}
	if finalized != 0 {
		t.Fatalf("expected genesis finalized height 0, got %d", finalized)
	}
}

func TestFinalizedHeightTracksAutomaticReorgBoundary(t *testing.T) {
	chain := NewChain()

	for i := uint64(0); i < params.MaxAutomaticReorgDepth+5; i++ {
		block := buildHistoryTestBlock(t, chain, 60)
		if err := chain.AddBlock(block); err != nil {
			t.Fatalf("failed adding block %d: %v", block.Height, err)
		}
	}

	finalized, err := FinalizedHeight(chain)
	if err != nil {
		t.Fatal(err)
	}
	if finalized != 5 {
		t.Fatalf("expected finalized height 5, got %d", finalized)
	}

	ok, err := IsBlockFinalized(chain, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected block 5 to be finalized")
	}

	ok, err = IsBlockFinalized(chain, 6)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected block 6 to remain inside automatic reorg window")
	}
}

func TestConfirmationsCountsBlockAndDescendants(t *testing.T) {
	chain := NewChain()

	block := buildHistoryTestBlock(t, chain, 60)
	if err := chain.AddBlock(block); err != nil {
		t.Fatal(err)
	}

	confirmations, err := Confirmations(chain, block.Height)
	if err != nil {
		t.Fatal(err)
	}
	if confirmations != 1 {
		t.Fatalf("expected 1 confirmation, got %d", confirmations)
	}
}

func TestFinalityRejectsHeightsAboveTip(t *testing.T) {
	chain := NewChain()

	if _, err := Confirmations(chain, 1); err == nil {
		t.Fatal("expected confirmations to reject height above tip")
	}
	if _, err := IsBlockFinalized(chain, 1); err == nil {
		t.Fatal("expected finality check to reject height above tip")
	}
}
