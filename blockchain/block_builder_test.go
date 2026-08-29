package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestNewBlockFromMempool(t *testing.T) {
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
		100*100_000_000,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	pool := mempool.NewMempool()

	if err := pool.AddTransaction(tx); err != nil {
		t.Fatal(err)
	}

	previous := NewGenesisBlock()

	block, err := NewBlockFromMempool(previous, pool)
	if err != nil {
		t.Fatal(err)
	}

	if block.Height != 1 {
		t.Fatalf("expected block height 1, got %d", block.Height)
	}

	if block.PreviousHash != previous.Hash() {
		t.Fatal("previous block hash does not match")
	}

	if len(block.Transactions) != 1 {
		t.Fatalf(
			"expected 1 transaction, got %d",
			len(block.Transactions),
		)
	}

	if block.MerkleRoot == "" {
		t.Fatal("Merkle root should not be empty")
	}

	if block.MerkleRoot == "Sudharma Network Empty Transaction Set" {
		t.Fatal("block should contain transaction data")
	}
}

func TestNewBlockFromMempoolWithPolicySelectsActivationVersion(t *testing.T) {
	policy := PoWPolicy{GPUV1ActivationHeight: 100}
	pool := mempool.NewMempool()

	before := NewGenesisBlock()
	before.Height = 98
	legacy, err := NewBlockFromMempoolWithPolicy(before, pool, policy)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Version != 1 {
		t.Fatalf("height 99 version = %d, want 1", legacy.Version)
	}

	before.Height = 99
	gpu, err := NewBlockFromMempoolWithPolicy(before, pool, policy)
	if err != nil {
		t.Fatal(err)
	}
	if gpu.Version != 2 {
		t.Fatalf("height 100 version = %d, want 2", gpu.Version)
	}
}
