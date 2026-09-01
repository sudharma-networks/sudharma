package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMainRejectsUnauthorizedMainnetNetwork(t *testing.T) {
	if params.MainnetLaunchAuthorized {
		t.Fatal("test requires MainnetLaunchAuthorized to remain false")
	}
	if _, err := params.ParseNetwork("mainnet"); err == nil {
		t.Fatal("expected sudharmad -network mainnet to be rejected while launch is unauthorized")
	}
}

func TestRunBlockMiningModeExitsAfterBoundedMining(t *testing.T) {
	calls := 0
	exitNodeLoop, err := runBlockMiningMode(1, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("runBlockMiningMode: %v", err)
	}
	if !exitNodeLoop {
		t.Fatal("bounded block-mining mode did not request node-loop exit")
	}
	if calls != 1 {
		t.Fatalf("bounded miner calls = %d, want 1", calls)
	}
}

func TestRunBlockMiningModeContinuesNormalNode(t *testing.T) {
	called := false
	exitNodeLoop, err := runBlockMiningMode(0, func() error {
		called = true
		return errors.New("must not run")
	})

	if err != nil {
		t.Fatalf("runBlockMiningMode: %v", err)
	}
	if exitNodeLoop {
		t.Fatal("normal node mode unexpectedly requested node-loop exit")
	}
	if called {
		t.Fatal("normal node mode unexpectedly ran bounded mining")
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
