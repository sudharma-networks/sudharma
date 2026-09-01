package mempool

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func signedPendingTransaction(
	t *testing.T,
	sender *wallet.Wallet,
	receiver string,
	nonce uint64,
) *transactions.Transaction {
	t.Helper()
	tx := transactions.NewTransaction(
		sender.Address,
		receiver,
		params.MinTransferAmount,
		nonce,
	)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestMempoolCachedBytesTrackAddRemoveAndClear(t *testing.T) {
	pool := NewMempool()
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx1 := signedPendingTransaction(t, sender, receiver.Address, 1)
	tx2 := signedPendingTransaction(t, sender, receiver.Address, 2)

	if err := pool.AddTransaction(tx1); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddTransaction(tx2); err != nil {
		t.Fatal(err)
	}
	want := tx1.EstimatedSerializedSize() + tx2.EstimatedSerializedSize()
	if got := pool.TotalEstimatedBytes(); got != want {
		t.Fatalf("cached bytes after add = %d, want %d", got, want)
	}

	pool.RemoveTransaction(tx1.ID)
	if got := pool.TotalEstimatedBytes(); got != tx2.EstimatedSerializedSize() {
		t.Fatalf("cached bytes after remove = %d, want %d", got, tx2.EstimatedSerializedSize())
	}
	if got := pool.CountForSender(sender.Address); got != 1 {
		t.Fatalf("sender count after remove = %d, want 1", got)
	}

	pool.Clear()
	if got := pool.TotalEstimatedBytes(); got != 0 {
		t.Fatalf("cached bytes after clear = %d, want 0", got)
	}
	if got := pool.Count(); got != 0 {
		t.Fatalf("mempool count after clear = %d, want 0", got)
	}
	if got := pool.CountForSender(sender.Address); got != 0 {
		t.Fatalf("sender count after clear = %d, want 0", got)
	}
}

func TestTransactionsForSenderIsIsolatedAndNonceOrdered(t *testing.T) {
	pool := NewMempool()
	senderA, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	senderB, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	for _, tx := range []*transactions.Transaction{
		signedPendingTransaction(t, senderA, receiver.Address, 2),
		signedPendingTransaction(t, senderB, receiver.Address, 1),
		signedPendingTransaction(t, senderA, receiver.Address, 1),
	} {
		if err := pool.AddTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}

	got := pool.TransactionsForSender(senderA.Address)
	if len(got) != 2 {
		t.Fatalf("sender A pending count = %d, want 2", len(got))
	}
	if got[0].From != senderA.Address || got[1].From != senderA.Address {
		t.Fatal("sender lookup leaked transaction from another sender")
	}
	if got[0].Nonce != 1 || got[1].Nonce != 2 {
		t.Fatalf("sender transactions not nonce ordered: %d, %d", got[0].Nonce, got[1].Nonce)
	}
}
