package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
)

func explorerTestChain() (*Chain, []*Block, []*transactions.Transaction) {
	chain := NewChain()

	tx1 := transactions.NewTransaction("alice", "bob", 100, 1)
	tx2 := transactions.NewTransaction("bob", "carol", 50, 1)
	tx3 := transactions.NewTransaction("alice", "dave", 25, 2)

	genesis := chain.Tip()
	b1 := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    genesis.Timestamp + 60,
		PreviousHash: genesis.Hash(),
		Difficulty:   1,
		Nonce:        1,
		MinerAddress: "miner-one",
		Transactions: []*transactions.Transaction{tx1, tx2},
	}
	b1.UpdateMerkleRoot()
	b2 := &Block{
		Version:      1,
		Height:       2,
		Timestamp:    b1.Timestamp + 60,
		PreviousHash: b1.Hash(),
		Difficulty:   1,
		Nonce:        2,
		MinerAddress: "miner-two",
		Transactions: []*transactions.Transaction{tx3},
	}
	b2.UpdateMerkleRoot()

	chain.mu.Lock()
	chain.blocks = []*Block{genesis, b1, b2}
	chain.mu.Unlock()

	return chain, []*Block{genesis, b1, b2}, []*transactions.Transaction{tx1, tx2, tx3}
}

func TestExplorerBlockByHash(t *testing.T) {
	chain, blocks, _ := explorerTestChain()

	got, ok := chain.BlockByHash(blocks[2].Hash())
	if !ok {
		t.Fatal("latest block was not found by hash")
	}
	if got.Height != 2 {
		t.Fatalf("height = %d, want 2", got.Height)
	}

	if _, ok := chain.BlockByHash("not-a-block-hash"); ok {
		t.Fatal("invalid block hash unexpectedly matched")
	}
}

func TestExplorerRecentBlocksNewestFirstAndBeforeExclusive(t *testing.T) {
	chain, _, _ := explorerTestChain()

	got := chain.RecentBlocks(2, nil)
	if len(got) != 2 || got[0].Height != 2 || got[1].Height != 1 {
		t.Fatalf("recent heights = %#v, want [2 1]", blockHeights(got))
	}

	before := uint64(2)
	got = chain.RecentBlocks(2, &before)
	if len(got) != 2 || got[0].Height != 1 || got[1].Height != 0 {
		t.Fatalf("before=2 heights = %#v, want [1 0]", blockHeights(got))
	}

	if got := chain.RecentBlocks(0, nil); len(got) != 0 {
		t.Fatalf("zero-limit blocks = %d, want 0", len(got))
	}
}

func TestExplorerRecentTransactionsCarryCanonicalBlockMetadata(t *testing.T) {
	chain, blocks, txs := explorerTestChain()

	got := chain.RecentTransactions(2, nil)
	if len(got) != 2 {
		t.Fatalf("recent transactions = %d, want 2", len(got))
	}
	if got[0].Transaction.ID != txs[2].ID || got[0].BlockHeight != 2 || got[0].BlockHash != blocks[2].Hash() {
		t.Fatalf("first recent transaction = %#v, want tx3 at block 2", got[0])
	}
	if got[1].Transaction.ID != txs[0].ID || got[1].BlockHeight != 1 || got[1].BlockHash != blocks[1].Hash() {
		t.Fatalf("second recent transaction = %#v, want tx1 at block 1", got[1])
	}

	before := uint64(2)
	got = chain.RecentTransactions(10, &before)
	if len(got) != 2 || got[0].BlockHeight != 1 || got[1].BlockHeight != 1 {
		t.Fatalf("before=2 transactions = %#v, want only block 1", got)
	}
}

func TestExplorerTransactionsForAddressMatchesSenderAndReceiver(t *testing.T) {
	chain, _, txs := explorerTestChain()

	got := chain.TransactionsForAddress("alice", 10, nil)
	if len(got) != 2 {
		t.Fatalf("alice history = %d, want 2", len(got))
	}
	if got[0].Transaction.ID != txs[2].ID || got[1].Transaction.ID != txs[0].ID {
		t.Fatalf("alice history IDs = %q, %q", got[0].Transaction.ID, got[1].Transaction.ID)
	}

	got = chain.TransactionsForAddress("bob", 10, nil)
	if len(got) != 2 {
		t.Fatalf("bob history = %d, want 2", len(got))
	}
	if got[0].Transaction.ID != txs[0].ID && got[0].Transaction.ID != txs[1].ID {
		t.Fatalf("unexpected bob history first ID %q", got[0].Transaction.ID)
	}

	if got := chain.TransactionsForAddress("", 10, nil); len(got) != 0 {
		t.Fatalf("empty-address history = %d, want 0", len(got))
	}
}

func blockHeights(blocks []*Block) []uint64 {
	heights := make([]uint64, 0, len(blocks))
	for _, block := range blocks {
		if block != nil {
			heights = append(heights, block.Height)
		}
	}
	return heights
}
