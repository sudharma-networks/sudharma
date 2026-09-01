package mempool

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMempoolRejectsAtTransactionCapacity(t *testing.T) {
	pool := NewMempool()
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < params.MaxMempoolTransactions; i++ {
		tx := transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			params.MinTransferAmount,
			uint64(i),
		)
		if err := tx.Sign(sender); err != nil {
			t.Fatal(err)
		}
		if err := pool.AddTransaction(tx); err != nil {
			t.Fatalf("seed transaction %d rejected: %v", i, err)
		}
	}

	overflow := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		params.MinTransferAmount,
		uint64(params.MaxMempoolTransactions),
	)
	if err := overflow.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddTransaction(overflow); err == nil {
		t.Fatal("transaction beyond capacity was accepted")
	}
}
