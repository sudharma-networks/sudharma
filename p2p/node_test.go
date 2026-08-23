package p2p

import (
	"testing"
	"time"
)

func TestTwoNodesPersistentHandshake(t *testing.T) {
	nodeA, err := NewNode(
		"node-a",
		"127.0.0.1:0",
		10,
		"tip-a",
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"node-b",
		"127.0.0.1:0",
		20,
		"tip-b",
	)

	if err != nil {
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

	peer, err := nodeA.Connect(
		nodeB.ListenAddress,
	)

	if err != nil {
		t.Fatal(err)
	}

	if peer.NodeID != "node-b" {
		t.Fatalf(
			"expected node-b, got %s",
			peer.NodeID,
		)
	}

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for nodeB.PeerCount() == 0 &&
		time.Now().Before(deadline) {

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if nodeA.PeerCount() != 1 {
		t.Fatalf(
			"expected node A peer count 1, got %d",
			nodeA.PeerCount(),
		)
	}

	if nodeB.PeerCount() != 1 {
		t.Fatalf(
			"expected node B peer count 1, got %d",
			nodeB.PeerCount(),
		)
	}

	// Connection should remain alive.
	time.Sleep(
		100 * time.Millisecond,
	)

	if nodeA.PeerCount() != 1 {
		t.Fatal(
			"node A persistent connection disappeared",
		)
	}

	if nodeB.PeerCount() != 1 {
		t.Fatal(
			"node B persistent connection disappeared",
		)
	}
}

func TestPersistentPing(t *testing.T) {
	nodeA, err := NewNode(
		"ping-node-a",
		"127.0.0.1:0",
		0,
		"tip-a",
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"ping-node-b",
		"127.0.0.1:0",
		0,
		"tip-b",
	)

	if err != nil {
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

	if _, err := nodeA.Connect(
		nodeB.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for nodeA.PeerCount() == 0 &&
		time.Now().Before(deadline) {

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if err := nodeA.SendPing(
		"ping-node-b",
		123456789,
	); err != nil {
		t.Fatalf(
			"failed to send ping: %v",
			err,
		)
	}

	// Give node B time to receive ping
	// and return pong.
	time.Sleep(
		100 * time.Millisecond,
	)

	// Both peer connections should still be alive.
	if nodeA.PeerCount() != 1 {
		t.Fatal(
			"node A disconnected after ping",
		)
	}

	if nodeB.PeerCount() != 1 {
		t.Fatal(
			"node B disconnected after ping",
		)
	}
}

func TestPingUnknownPeerFails(t *testing.T) {
	node, err := NewNode(
		"lonely-node",
		"127.0.0.1:0",
		0,
		"tip",
	)

	if err != nil {
		t.Fatal(err)
	}

	if err := node.SendPing(
		"missing-peer",
		1,
	); err == nil {
		t.Fatal(
			"ping to unknown peer was accepted",
		)
	}
}
