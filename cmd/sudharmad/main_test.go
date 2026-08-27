package main

import (
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestRunBlockMiningTestConfirmsMempoolTransaction(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	minerWallet, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	initialBalance := uint64(50) * params.CoinDecimals
	if err := state.Credit(sender.Address, initialBalance); err != nil {
		t.Fatal(err)
	}

	node, err := p2p.NewNode("block-mining-test", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}

	amount := uint64(10) * params.CoinDecimals
	tx := transactions.NewTransaction(sender.Address, receiver.Address, amount, 1)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if err := blockchain.ValidateMempoolTransaction(state, node.Mempool().AllTransactions(), tx); err != nil {
		t.Fatalf("valid transaction rejected: %v", err)
	}
	if err := node.Mempool().AddTransaction(tx); err != nil {
		t.Fatal(err)
	}

	beforeHeight := chain.Height()
	tempDir := t.TempDir()
	if err := runBlockMiningTest(
		chain,
		state,
		node,
		filepath.Join(tempDir, "chain.json"),
		filepath.Join(tempDir, "state.json"),
		minerWallet.Address,
		1,
	); err != nil {
		t.Fatalf("transaction-confirming mining failed: %v", err)
	}

	if got := chain.Height(); got != beforeHeight+1 {
		t.Fatalf("expected height %d, got %d", beforeHeight+1, got)
	}
	if got := state.Balance(receiver.Address); got != amount {
		t.Fatalf("expected receiver balance %d, got %d", amount, got)
	}
	if got := node.Mempool().Count(); got != 0 {
		t.Fatalf("expected empty mempool after mining, got %d transaction(s)", got)
	}
	if _, _, found := chain.TransactionByID(tx.ID); !found {
		t.Fatalf("mined transaction %s was not indexed in the chain", tx.ID)
	}
}

func TestRunBlockMiningTestRejectsUnsafeCounts(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := p2p.NewNode("block-mining-limits", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}

	for _, count := range []uint64{0, 101} {
		if err := runBlockMiningTest(chain, state, node, "chain.json", "state.json", "miner", count); err == nil {
			t.Fatalf("expected count %d to be rejected", count)
		}
	}
}

func TestMineBlocksRequestIsOneShot(t *testing.T) {
	if !mineBlocksIsOneShot(1) {
		t.Fatal("expected -mineblocks 1 to exit after bounded mining completes")
	}
	if mineBlocksIsOneShot(0) {
		t.Fatal("expected ordinary node mode to continue into the main loop")
	}
}
