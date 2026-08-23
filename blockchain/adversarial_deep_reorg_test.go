package blockchain

import (
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestAdversarialHigherWorkDeepForkCannotCrossFinalityBoundary(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	// Build two valid chains that diverge immediately after genesis. The
	// candidate is one block longer, so it has greater valid cumulative work,
	// but adopting it would replace more history than the automatic-finality
	// policy permits.
	for i := uint64(0); i < params.MaxAutomaticReorgDepth+1; i++ {
		block := buildHistoryTestBlock(t, current, 60)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}
	for i := uint64(0); i < params.MaxAutomaticReorgDepth+2; i++ {
		block := buildHistoryTestBlock(t, candidate, 61)
		if err := candidate.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	if candidate.TotalWork().Cmp(current.TotalWork()) <= 0 {
		t.Fatal("test setup invalid: candidate must have greater cumulative work")
	}

	originalTip := current.Tip().Hash()
	adopted, err := ReorganizeToCandidate(current, NewState(), candidate)
	if err == nil {
		t.Fatal("expected deep higher-work fork to be rejected")
	}
	if adopted {
		t.Fatal("deep higher-work fork was adopted")
	}
	if current.Tip().Hash() != originalTip {
		t.Fatal("current chain changed after rejected deep fork")
	}
}

func TestAdversarialInvalidHigherWorkCandidateRejectedBeforeForkChoice(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	currentBlock := buildHistoryTestBlock(t, current, 60)
	if err := current.AddBlock(currentBlock); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		block := buildHistoryTestBlock(t, candidate, 61)
		if err := candidate.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	if candidate.TotalWork().Cmp(current.TotalWork()) <= 0 {
		t.Fatal("test setup invalid: candidate must initially have greater work")
	}

	// Tamper with an already accepted candidate block after cumulative work was
	// cached. The candidate still advertises higher work, but revalidation must
	// reject it before fork choice can use that metadata.
	candidate.mu.Lock()
	candidate.blocks[1].PreviousHash = "attacker-forged-parent"
	candidate.mu.Unlock()

	originalTip := current.Tip().Hash()
	adopted, err := ReorganizeToCandidate(current, NewState(), candidate)
	if err == nil {
		t.Fatal("expected tampered higher-work candidate to be rejected")
	}
	if !strings.Contains(err.Error(), "candidate chain validation failed") {
		t.Fatalf("expected candidate validation failure, got: %v", err)
	}
	if adopted {
		t.Fatal("tampered higher-work candidate was adopted")
	}
	if current.Tip().Hash() != originalTip {
		t.Fatal("current chain changed after invalid candidate rejection")
	}
}

func TestAutomaticReorgDepthBoundaryIsExact(t *testing.T) {
	if err := ValidateAutomaticReorgDepth(params.MaxAutomaticReorgDepth); err != nil {
		t.Fatalf("maximum permitted automatic reorg depth was rejected: %v", err)
	}
	if err := ValidateAutomaticReorgDepth(params.MaxAutomaticReorgDepth + 1); err == nil {
		t.Fatal("expected reorg one block beyond the automatic limit to be rejected")
	}
}
