package blockchain

import (
	"testing"
	"time"
)

func TestOldTimestampRejected(t *testing.T) {
	previous := NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    previous.Timestamp,
		PreviousHash: previous.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if err := ValidateBlockBasic(
		block,
		previous,
	); err == nil {
		t.Fatal(
			"block with non-increasing timestamp was accepted",
		)
	}
}

func TestFutureTimestampRejected(t *testing.T) {
	previous := NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    time.Now().Unix() + MaxFutureBlockSeconds + 60,
		PreviousHash: previous.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	if err := ValidateBlockBasic(
		block,
		previous,
	); err == nil {
		t.Fatal(
			"block too far in future was accepted",
		)
	}
}
