package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionBroadcastBetweenNodes(t *testing.T) {
	nodeA, err := NewNode(
		"tx-node-a",
		"127.0.0.1:0",
		0,
		"tip-a",
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"tx-node-b",
		"127.0.0.1:0",
		0,
		"tip-b",
	)

	if err != nil {
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

	development, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	// Node A's confirmed blockchain state.
	stateA := blockchain.NewState()

	stateA.SetDevelopmentAddress(
		development.Address,
	)

	// Fund sender with 100 SUDH.
	if err := stateA.Credit(
		sender.Address,
		100*params.CoinDecimals,
	); err != nil {
		t.Fatal(err)
	}

	if err := nodeA.SetState(
		stateA,
	); err != nil {
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

	if _, err := nodeB.Connect(
		nodeA.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.BroadcastTransaction(
		tx,
	); err != nil {
		t.Fatal(err)
	}

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for nodeA.MempoolCount() == 0 &&
		time.Now().Before(deadline) {

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if nodeA.MempoolCount() != 1 {
		t.Fatalf(
			"expected node A mempool count 1, got %d",
			nodeA.MempoolCount(),
		)
	}

	received, ok :=
		nodeA.MempoolTransaction(
			tx.ID,
		)

	if !ok {
		t.Fatal(
			"node A did not receive valid transaction",
		)
	}

	if received.ID != tx.ID {
		t.Fatal(
			"received transaction ID does not match",
		)
	}

	if !received.Verify() {
		t.Fatal(
			"received transaction signature is invalid",
		)
	}
}

func TestUnfundedTransactionRejectedByReceivingNode(t *testing.T) {
	nodeA, err := NewNode(
		"reject-node-a",
		"127.0.0.1:0",
		0,
		"tip-a",
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"reject-node-b",
		"127.0.0.1:0",
		0,
		"tip-b",
	)

	if err != nil {
		t.Fatal(err)
	}

	development, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	stateA := blockchain.NewState()

	stateA.SetDevelopmentAddress(
		development.Address,
	)

	if err := nodeA.SetState(
		stateA,
	); err != nil {
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

	if _, err := nodeB.Connect(
		nodeA.ListenAddress,
	); err != nil {
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

	// Sender owns zero SUDH.
	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.BroadcastTransaction(
		tx,
	); err != nil {
		t.Fatal(err)
	}

	time.Sleep(
		100 * time.Millisecond,
	)

	if nodeA.MempoolCount() != 0 {
		t.Fatalf(
			"unfunded transaction entered mempool; count %d",
			nodeA.MempoolCount(),
		)
	}
}
