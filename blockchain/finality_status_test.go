package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestCurrentFinalityStatusAtGenesis(t *testing.T) {
	chain := NewChain()

	status, err := CurrentFinalityStatus(chain)
	if err != nil {
		t.Fatal(err)
	}
	if status.TipHeight != 0 || status.FinalizedHeight != 0 || status.ReorgWindow != 0 {
		t.Fatalf("unexpected genesis finality status: %+v", status)
	}
}

func TestCurrentFinalityStatusTracksBoundary(t *testing.T) {
	chain := NewChain()

	for i := uint64(0); i < params.MaxAutomaticReorgDepth+3; i++ {
		block := buildHistoryTestBlock(t, chain, 60)
		if err := chain.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	status, err := CurrentFinalityStatus(chain)
	if err != nil {
		t.Fatal(err)
	}
	if status.FinalizedHeight != 3 {
		t.Fatalf("expected finalized height 3, got %d", status.FinalizedHeight)
	}
	if status.ReorgWindow != params.MaxAutomaticReorgDepth {
		t.Fatalf("expected reorg window %d, got %d", params.MaxAutomaticReorgDepth, status.ReorgWindow)
	}
}

func TestFinalityStatusForBlockUsesConsensusHelpers(t *testing.T) {
	chain := NewChain()

	for i := uint64(0); i < params.MaxAutomaticReorgDepth+2; i++ {
		block := buildHistoryTestBlock(t, chain, 60)
		if err := chain.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	status, err := FinalityStatusForBlock(chain, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Finalized {
		t.Fatal("expected block 1 to be finalized")
	}
	if status.Confirmations != chain.Height() {
		t.Fatalf("expected %d confirmations, got %d", chain.Height(), status.Confirmations)
	}
}

func TestFinalityStatusRejectsNilAndAboveTip(t *testing.T) {
	if _, err := CurrentFinalityStatus(nil); err == nil {
		t.Fatal("expected nil chain to be rejected")
	}

	chain := NewChain()
	if _, err := FinalityStatusForBlock(chain, 1); err == nil {
		t.Fatal("expected above-tip block height to be rejected")
	}
}
