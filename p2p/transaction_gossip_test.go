package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionGossipThreeNodes(t *testing.T) {

	nodeA, err := NewNode(
		"gossip-node-a",
		"127.0.0.1:0",
		0,
		"tip-a",
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"gossip-node-b",
		"127.0.0.1:0",
		0,
		"tip-b",
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeC, err := NewNode(
		"gossip-node-c",
		"127.0.0.1:0",
		0,
		"tip-c",
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

	// All three nodes need equivalent confirmed state so that
	// every receiving node can independently validate the TX.
	stateA := blockchain.NewState()
	stateB := blockchain.NewState()
	stateC := blockchain.NewState()

	stateA.SetDevelopmentAddress(development.Address)
	stateB.SetDevelopmentAddress(development.Address)
	stateC.SetDevelopmentAddress(development.Address)

	for _, state := range []*blockchain.State{
		stateA,
		stateB,
		stateC,
	} {

		if err := state.Credit(
			sender.Address,
			100*params.CoinDecimals,
		); err != nil {
			t.Fatal(err)
		}
	}

	if err := nodeA.SetState(stateA); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.SetState(stateB); err != nil {
		t.Fatal(err)
	}

	if err := nodeC.SetState(stateC); err != nil {
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

	if err := nodeC.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeC.Stop()

	// Build a line topology:
	//
	// A <----> B <----> C
	//
	// A and C are deliberately NOT directly connected.

	if _, err := nodeA.Connect(
		nodeB.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeB.Connect(
		nodeC.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if nodeA.PeerCount() != 1 {
		t.Fatalf(
			"expected node A to have 1 peer, got %d",
			nodeA.PeerCount(),
		)
	}

	if nodeC.PeerCount() != 1 {
		t.Fatalf(
			"expected node C to have 1 peer, got %d",
			nodeC.PeerCount(),
		)
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

	if !tx.Verify() {
		t.Fatal(
			"created transaction failed signature verification",
		)
	}

	// A broadcasts only to its direct peer B.
	// B must validate, accept and relay it to C.
	if err := nodeA.BroadcastTransaction(
		tx,
	); err != nil {
		t.Fatal(err)
	}

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for time.Now().Before(deadline) {

		_, bHas :=
			nodeB.MempoolTransaction(
				tx.ID,
			)

		_, cHas :=
			nodeC.MempoolTransaction(
				tx.ID,
			)

		if bHas && cHas {
			break
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if _, ok :=
		nodeB.MempoolTransaction(
			tx.ID,
		); !ok {

		t.Fatal(
			"node B did not accept transaction",
		)
	}

	if _, ok :=
		nodeC.MempoolTransaction(
			tx.ID,
		); !ok {

		t.Fatal(
			"node C did not receive transaction through node B gossip",
		)
	}

	if nodeB.MempoolCount() != 1 {
		t.Fatalf(
			"expected node B mempool count 1, got %d",
			nodeB.MempoolCount(),
		)
	}

	if nodeC.MempoolCount() != 1 {
		t.Fatalf(
			"expected node C mempool count 1, got %d",
			nodeC.MempoolCount(),
		)
	}

	// Wait briefly to detect accidental relay loops.
	time.Sleep(
		200 * time.Millisecond,
	)

	if nodeB.MempoolCount() != 1 {
		t.Fatalf(
			"duplicate gossip changed node B mempool count: %d",
			nodeB.MempoolCount(),
		)
	}

	if nodeC.MempoolCount() != 1 {
		t.Fatalf(
			"duplicate gossip changed node C mempool count: %d",
			nodeC.MempoolCount(),
		)
	}
}
