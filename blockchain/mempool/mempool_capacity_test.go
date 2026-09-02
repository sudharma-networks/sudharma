package mempool

import (
	"fmt"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

func TestMempoolRejectsAtTransactionCapacity(t *testing.T) {
	pool := NewMempool()
	receiver := "ffffffffffffffffffffffffffffffffffffffff"

	added := 0
	for senderIndex := 0; added < params.MaxMempoolTransactions; senderIndex++ {
		sender := fmt.Sprintf("%040x", senderIndex+1)
		for nonce := uint64(1); nonce <= uint64(params.MaxMempoolTransactionsPerSender) && added < params.MaxMempoolTransactions; nonce++ {
			tx := transactions.NewTransaction(
				sender,
				receiver,
				params.MinTransferAmount,
				nonce,
			)
			if err := pool.AddTransaction(tx); err != nil {
				t.Fatalf("seed transaction %d rejected: %v", added, err)
			}
			added++
		}
	}

	// Use a fresh sender so the global count limit, not the per-sender policy,
	// is what rejects the next transaction.
	overflow := transactions.NewTransaction(
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		receiver,
		params.MinTransferAmount,
		1,
	)
	if err := pool.AddTransaction(overflow); err == nil {
		t.Fatal("transaction beyond global capacity was accepted")
	}
}
