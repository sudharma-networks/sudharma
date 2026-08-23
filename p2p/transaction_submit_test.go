package p2p

import (
	"sync"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestConcurrentLocalSubmissionsRejectSameNonceConflict(t *testing.T) {
	chain := blockchain.NewChain()
	state := blockchain.NewState()
	node, err := NewNode("submit-concurrency", "127.0.0.1:0", chain.Height(), chain.Tip().Hash())
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
	receiverA, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiverB, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(sender.Address, 10*params.CoinDecimals); err != nil {
		t.Fatal(err)
	}

	txA := transactions.NewTransaction(sender.Address, receiverA.Address, params.CoinDecimals, 1)
	if err := txA.Sign(sender); err != nil {
		t.Fatal(err)
	}
	txB := transactions.NewTransaction(sender.Address, receiverB.Address, params.CoinDecimals, 1)
	if err := txB.Sign(sender); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, tx := range []*transactions.Transaction{txA, txB} {
		wg.Add(1)
		go func(tx *transactions.Transaction) {
			defer wg.Done()
			<-start
			_, err := node.SubmitTransaction(tx)
			results <- err
		}(tx)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	failures := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("expected one accepted and one rejected submission, got successes=%d failures=%d", successes, failures)
	}
	if node.MempoolCount() != 1 {
		t.Fatalf("expected exactly one pending transaction, got %d", node.MempoolCount())
	}
}
