package p2p

import (
	"testing"
	"time"
)

func TestPeerDiscoveryAutoConnectThreeNodes(t *testing.T) {

	nodeA, err := NewNode(
		"auto-discovery-a",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"auto-discovery-b",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeC, err := NewNode(
		"auto-discovery-c",
		"127.0.0.1:0",
		0,
		"",
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

	if err := nodeC.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeC.Stop()

	if _, err := nodeB.Connect(
		nodeC.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeA.Connect(
		nodeB.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if nodeA.PeerCount() != 1 {
		t.Fatalf(
			"expected A to begin with 1 peer, got %d",
			nodeA.PeerCount(),
		)
	}

	if err := nodeA.SendGetPeers(
		"auto-discovery-b",
	); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(
		3 * time.Second,
	)

	for time.Now().Before(deadline) {

		if nodeA.PeerCount() == 2 &&
			nodeC.PeerCount() == 2 {

			return
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	t.Fatalf(
		"automatic discovered-peer connection failed: A peers=%d C peers=%d",
		nodeA.PeerCount(),
		nodeC.PeerCount(),
	)
}

func TestAutoConnectDiscoveredPeerFailureIsNonFatal(t *testing.T) {

	node, err := NewNode(
		"auto-discovery-local",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	added := node.mergeDiscoveredPeers(
		[]KnownPeer{
			{
				NodeID:  "offline-peer",
				Address: "127.0.0.1:1",
			},
		},
	)

	if added != 1 {
		t.Fatalf(
			"expected 1 discovered peer, got %d",
			added,
		)
	}

	connected, failed :=
		node.AutoConnectDiscoveredPeers()

	if connected != 0 {
		t.Fatalf(
			"expected 0 successful connections, got %d",
			connected,
		)
	}

	if failed != 1 {
		t.Fatalf(
			"expected 1 failed connection, got %d",
			failed,
		)
	}

	if node.PeerCount() != 0 {
		t.Fatalf(
			"failed discovered peer changed peer count to %d",
			node.PeerCount(),
		)
	}
}
