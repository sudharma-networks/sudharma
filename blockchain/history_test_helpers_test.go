package blockchain

import "testing"

func buildHistoryTestBlock(t *testing.T, chain *Chain, blockTime int64) *Block {
	t.Helper()

	if chain == nil {
		t.Fatal("chain cannot be nil")
	}

	previous := chain.Tip()
	if previous == nil {
		t.Fatal("chain tip cannot be nil")
	}

	difficulty, err := ExpectedNextDifficulty(chain)
	if err != nil {
		t.Fatalf("failed calculating expected difficulty: %v", err)
	}

	block := &Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + blockTime,
		PreviousHash: previous.Hash(),
		Difficulty:   difficulty,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()
	if !mineTestBlock(block, 1_000_000) {
		t.Fatal("failed to mine history-aware test block")
	}

	return block
}
