package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func mineTestBlock(block *Block, maxAttempts uint64) bool {
	for nonce := uint64(0); nonce < maxAttempts; nonce++ {
		block.Nonce = nonce

		if validBlockProofOfWork(block) {
			return true
		}
	}

	return false
}

func TestValidBlockAccepted(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	previous := NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    previous.Timestamp + 60,
		PreviousHash: previous.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: []*transactions.Transaction{
			tx,
		},
	}

	block.UpdateMerkleRoot()

	if !mineTestBlock(block, 1_000_000) {
		t.Fatal("failed to mine test block")
	}

	if err := ValidateBlockBasic(
		block,
		previous,
	); err != nil {
		t.Fatalf(
			"valid block was rejected: %v",
			err,
		)
	}
}

func TestWrongPreviousHashRejected(t *testing.T) {
	previous := NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    previous.Timestamp + 60,
		PreviousHash: "wrong-hash",
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
			"block with wrong previous hash was accepted",
		)
	}
}

func TestWrongHeightRejected(t *testing.T) {
	previous := NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       5,
		Timestamp:    previous.Timestamp + 60,
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
			"block with wrong height was accepted",
		)
	}
}

func TestTamperedMerkleRootRejected(t *testing.T) {
	previous := NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    previous.Timestamp + 60,
		PreviousHash: previous.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	block.MerkleRoot = "tampered-merkle-root"

	if err := ValidateBlockBasic(
		block,
		previous,
	); err == nil {
		t.Fatal(
			"block with tampered merkle root was accepted",
		)
	}
}
