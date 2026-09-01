package mempool

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMempoolRejectsPerSenderQueueBeyondBound(t *testing.T) {
	const maxPerSender = 64

	pool := NewMempool()
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	for nonce := uint64(1); nonce <= maxPerSender; nonce++ {
		tx := transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			params.MinTransferAmount,
			nonce,
		)
		if err := tx.Sign(sender); err != nil {
			t.Fatal(err)
		}
		if err := pool.AddTransaction(tx); err != nil {
			t.Fatalf("transaction %d within per-sender bound rejected: %v", nonce, err)
		}
	}

	overflow := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		params.MinTransferAmount,
		maxPerSender+1,
	)
	if err := overflow.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddTransaction(overflow); err == nil {
		t.Fatal("transaction beyond per-sender pending bound was accepted")
	}
}
