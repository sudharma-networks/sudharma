package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMempoolsSynchronizeAfterExplicitRequest(t *testing.T) {
	senderA, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiverA, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	senderB, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiverB, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	stateA := blockchain.NewState()
	stateB := blockchain.NewState()

	balance := uint64(100) * params.CoinDecimals

	for _, state := range []*blockchain.State{stateA, stateB} {
		if err := state.Credit(senderA.Address, balance); err != nil {
			t.Fatal(err)
		}

		if err := state.Credit(senderB.Address, balance); err != nil {
			t.Fatal(err)
		}
	}

	chainA := blockchain.NewChain()
	chainB := blockchain.NewChain()

	nodeA, err := NewNode(
		"mempool-explicit-a",
		"127.0.0.1:0",
		chainA.Height(),
		chainA.Tip().Hash(),
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"mempool-explicit-b",
		"127.0.0.1:0",
		chainB.Height(),
		chainB.Tip().Hash(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeA.SetChain(chainA); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.SetChain(chainB); err != nil {
		t.Fatal(err)
	}
	if err := nodeA.SetState(stateA); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.SetState(stateB); err != nil {
		t.Fatal(err)
	}

	txA := transactions.NewTransaction(
		senderA.Address,
		receiverA.Address,
		10*params.CoinDecimals,
		1,
	)
	if err := txA.Sign(senderA); err != nil {
		t.Fatal(err)
	}

	if err := blockchain.ValidateMempoolTransaction(
		stateA,
		nodeA.Mempool().AllTransactions(),
		txA,
	); err != nil {
		t.Fatal(err)
	}

	if err := nodeA.Mempool().AddTransaction(txA); err != nil {
		t.Fatal(err)
	}

	txB := transactions.NewTransaction(
		senderB.Address,
		receiverB.Address,
		5*params.CoinDecimals,
		1,
	)
	if err := txB.Sign(senderB); err != nil {
		t.Fatal(err)
	}

	if err := blockchain.ValidateMempoolTransaction(
		stateB,
		nodeB.Mempool().AllTransactions(),
		txB,
	); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.Mempool().AddTransaction(txB); err != nil {
		t.Fatal(err)
	}

	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeA.Stop()

	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeB.Stop()

	peer, err := nodeB.Connect(nodeA.ListenAddress)
	if err != nil {
		t.Fatal(err)
	}

	// Connection alone must no longer push mempools.
	time.Sleep(50 * time.Millisecond)

	if nodeA.MempoolCount() != 1 {
		t.Fatalf(
			"node A mempool changed before explicit sync: %d",
			nodeA.MempoolCount(),
		)
	}

	if nodeB.MempoolCount() != 1 {
		t.Fatalf(
			"node B mempool changed before explicit sync: %d",
			nodeB.MempoolCount(),
		)
	}

	if err := nodeB.SyncMempoolWithPeer(peer.NodeID); err != nil {
		t.Fatal(err)
	}

	// SyncMempoolWithPeer is a sequencing barrier for callers that need the
	// peer's pending transactions immediately after chain sync, such as the
	// one-shot transaction-confirming miner. When it returns, both snapshots
	// must already have been processed.
	if _, ok := nodeA.MempoolTransaction(txB.ID); !ok {
		t.Fatal("node A did not receive node B pending transaction before sync returned")
	}

	if _, ok := nodeB.MempoolTransaction(txA.ID); !ok {
		t.Fatal("node B did not receive node A pending transaction before sync returned")
	}

	if nodeA.MempoolCount() != 2 {
		t.Fatalf(
			"expected node A mempool count 2, got %d",
			nodeA.MempoolCount(),
		)
	}

	if nodeB.MempoolCount() != 2 {
		t.Fatalf(
			"expected node B mempool count 2, got %d",
			nodeB.MempoolCount(),
		)
	}
}

func TestGetMempoolMessageEncodeDecode(t *testing.T) {
	data, err := NewGetMempoolMessage()
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if message.Type != MessageGetMempool {
		t.Fatalf(
			"expected %s, got %s",
			MessageGetMempool,
			message.Type,
		)
	}

	if err := DecodeGetMempool(message); err != nil {
		t.Fatal(err)
	}
}
