package blockchain

import "testing"

func TestAdversarialTimestampRegressionRejected(t *testing.T) {
	chain := NewChain()
	block := buildHistoryTestBlock(t, chain, 60)

	previous := chain.Tip()
	block.Timestamp = previous.Timestamp

	if err := chain.AddBlock(block); err == nil {
		t.Fatal("expected non-increasing timestamp attack to be rejected")
	}
}

func TestAdversarialDifficultySpoofRejected(t *testing.T) {
	chain := NewChain()
	block := buildHistoryTestBlock(t, chain, 60)

	block.Difficulty++

	if err := chain.AddBlock(block); err == nil {
		t.Fatal("expected forged difficulty to be rejected")
	}
}

func TestAdversarialPreviousHashTamperingRejected(t *testing.T) {
	chain := NewChain()
	block := buildHistoryTestBlock(t, chain, 60)

	block.PreviousHash = "00deadbeef"

	if err := chain.AddBlock(block); err == nil {
		t.Fatal("expected previous-hash tampering to be rejected")
	}
}

func TestAdversarialHeightSkipRejected(t *testing.T) {
	chain := NewChain()
	block := buildHistoryTestBlock(t, chain, 60)

	block.Height++

	if err := chain.AddBlock(block); err == nil {
		t.Fatal("expected skipped-height block to be rejected")
	}
}

func TestAdversarialMerkleTamperingRejected(t *testing.T) {
	chain := NewChain()
	block := buildHistoryTestBlock(t, chain, 60)

	block.MerkleRoot = "tampered-merkle-root"

	if err := chain.AddBlock(block); err == nil {
		t.Fatal("expected merkle-root tampering to be rejected")
	}
}
