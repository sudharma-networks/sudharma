package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestWaitForPendingTransactionsObservesAsyncMempoolArrival(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := p2p.NewNode("mempool-wait-test", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(sender.Address, 50*params.CoinDecimals); err != nil {
		t.Fatal(err)
	}
	tx := transactions.NewTransaction(sender.Address, receiver.Address, 10*params.CoinDecimals, 1)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(25 * time.Millisecond)
		_ = node.Mempool().AddTransaction(tx)
	}()

	if !waitForPendingTransactions(node, 250*time.Millisecond) {
		t.Fatal("expected asynchronous pending transaction before timeout")
	}
	if got := node.Mempool().Count(); got != 1 {
		t.Fatalf("expected 1 pending transaction, got %d", got)
	}
}

func TestWaitForPendingTransactionsTimesOutWhenMempoolStaysEmpty(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := p2p.NewNode("mempool-empty-test", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	if waitForPendingTransactions(node, 40*time.Millisecond) {
		t.Fatal("did not expect a pending transaction")
	}
	if elapsed := time.Since(started); elapsed < 30*time.Millisecond {
		t.Fatalf("wait returned too early after %s", elapsed)
	}
}

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
