package blockchain

import "testing"

func TestGenesisBlock(t *testing.T) {
	block := NewGenesisBlock()

	if block.Height != 0 {
		t.Fatalf("expected genesis height 0, got %d", block.Height)
	}

	if block.PreviousHash != "0" {
		t.Fatalf("expected previous hash 0, got %s", block.PreviousHash)
	}

	if block.Hash() == "" {
		t.Fatal("genesis block hash should not be empty")
	}
}
