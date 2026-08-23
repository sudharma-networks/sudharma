package blockchain

import "testing"

func TestShorterHigherWorkChainWins(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	// Build a longer weak chain using slow blocks.
	for i := 0; i < 6; i++ {
		block := buildHistoryTestBlock(t, current, 180)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	// Build a shorter but stronger chain using fast blocks.
	for i := 0; i < 4; i++ {
		block := buildHistoryTestBlock(t, candidate, 10)
		if err := candidate.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	if candidate.Height() >= current.Height() {
		t.Fatal("candidate must be shorter for this test")
	}

	if candidate.TotalWork().Cmp(current.TotalWork()) <= 0 {
		t.Fatalf(
			"test setup invalid: candidate work %s must exceed current work %s",
			candidate.TotalWork().String(),
			current.TotalWork().String(),
		)
	}

	best, err := BetterChain(current, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if best != candidate {
		t.Fatal("shorter chain with greater cumulative work was not selected")
	}
}
