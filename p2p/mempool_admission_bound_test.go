package p2p

import (
	"fmt"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func newAdmissionTestNode(t *testing.T) (*Node, *blockchain.State) {
	t.Helper()
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := NewNode("admission-bound", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := node.SetChain(chain); err != nil {
		t.Fatal(err)
	}
	if err := node.SetState(state); err != nil {
		t.Fatal(err)
	}
	return node, state
}

func TestLocalSubmissionIsBoundedByPerSenderQueue(t *testing.T) {
	node, state := newAdmissionTestNode(t)
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(sender.Address, params.CoinDecimals); err != nil {
		t.Fatal(err)
	}

	for nonce := uint64(1); nonce <= uint64(params.MaxMempoolTransactionsPerSender); nonce++ {
		tx := transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			params.MinTransferAmount,
			nonce,
		)
		if err := tx.Sign(sender); err != nil {
			t.Fatal(err)
		}
		if _, err := node.SubmitTransaction(tx); err != nil {
			t.Fatalf("sender transaction %d rejected before bound: %v", nonce, err)
		}
	}

	overflow := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		params.MinTransferAmount,
		uint64(params.MaxMempoolTransactionsPerSender)+1,
	)
	if err := overflow.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if _, err := node.SubmitTransaction(overflow); err == nil {
		t.Fatal("transaction beyond per-sender admission bound was accepted")
	}
	if got := node.Mempool().CountForSender(sender.Address); got != params.MaxMempoolTransactionsPerSender {
		t.Fatalf("sender pending count = %d, want %d", got, params.MaxMempoolTransactionsPerSender)
	}
}

func TestLocalSubmissionDoesNotReplayLargeUnrelatedPool(t *testing.T) {
	node, state := newAdmissionTestNode(t)
	receiver := "ffffffffffffffffffffffffffffffffffffffff"

	// Seed hundreds of structurally valid but unsigned/stale entries from other
	// senders. They model unrelated mempool state that must never be replayed
	// while validating a different sender.
	const unrelatedSenders = 8
	for senderIndex := 0; senderIndex < unrelatedSenders; senderIndex++ {
		sender := fmt.Sprintf("%040x", senderIndex+1)
		for nonce := uint64(1); nonce <= uint64(params.MaxMempoolTransactionsPerSender); nonce++ {
			tx := transactions.NewTransaction(sender, receiver, params.MinTransferAmount, nonce)
			if err := node.Mempool().AddTransaction(tx); err != nil {
				t.Fatalf("failed seeding unrelated entry sender=%d nonce=%d: %v", senderIndex, nonce, err)
			}
		}
	}

	target, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	targetReceiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(target.Address, params.CoinDecimals); err != nil {
		t.Fatal(err)
	}

	candidate := transactions.NewTransaction(
		target.Address,
		targetReceiver.Address,
		params.MinTransferAmount,
		1,
	)
	if err := candidate.Sign(target); err != nil {
		t.Fatal(err)
	}
	if _, err := node.SubmitTransaction(candidate); err != nil {
		t.Fatalf("valid target transaction was coupled to unrelated pool: %v", err)
	}
	if got := node.Mempool().CountForSender(target.Address); got != 1 {
		t.Fatalf("target sender pending count = %d, want 1", got)
	}
}
