package blockchain

import "testing"

func TestReorgDepthSameChainIsZero(t *testing.T) {
	chain := NewChain()

	depth, common, err := ReorgDepth(chain, chain)
	if err != nil {
		t.Fatal(err)
	}
	if depth != 0 {
		t.Fatalf("expected depth 0, got %d", depth)
	}
	if common != chain.Height() {
		t.Fatalf("expected common height %d, got %d", chain.Height(), common)
	}
}

func TestReorgDepthOneBlockReplacement(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	shared := buildHistoryTestBlock(t, current, 60)
	if err := current.AddBlock(shared); err != nil {
		t.Fatal(err)
	}
	if err := candidate.AddBlock(shared); err != nil {
		t.Fatal(err)
	}

	currentBlock := buildHistoryTestBlock(t, current, 60)
	if err := current.AddBlock(currentBlock); err != nil {
		t.Fatal(err)
	}

	candidateBlock := buildHistoryTestBlock(t, candidate, 10)
	if err := candidate.AddBlock(candidateBlock); err != nil {
		t.Fatal(err)
	}

	depth, common, err := ReorgDepth(current, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if depth != 1 {
		t.Fatalf("expected depth 1, got %d", depth)
	}
	if common != 1 {
		t.Fatalf("expected common height 1, got %d", common)
	}
}

func TestReorgDepthMultipleBlocks(t *testing.T) {
	current := NewChain()
	candidate := NewChain()

	shared := buildHistoryTestBlock(t, current, 60)
	if err := current.AddBlock(shared); err != nil {
		t.Fatal(err)
	}
	if err := candidate.AddBlock(shared); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		block := buildHistoryTestBlock(t, current, 60)
		if err := current.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 2; i++ {
		block := buildHistoryTestBlock(t, candidate, 10)
		if err := candidate.AddBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	depth, common, err := ReorgDepth(current, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if depth != 3 {
		t.Fatalf("expected depth 3, got %d", depth)
	}
	if common != 1 {
		t.Fatalf("expected common height 1, got %d", common)
	}
}

func TestReorgDepthNilCandidateRejected(t *testing.T) {
	current := NewChain()
	_, _, err := ReorgDepth(current, nil)
	if err == nil {
		t.Fatal("expected nil candidate to be rejected")
	}
}
