package p2p

import (
	"testing"
	"time"
)

func waitForPeerCount(t *testing.T, node *Node, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if node.PeerCount() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected peer count %d, got %d", want, node.PeerCount())
}

func TestStopClosesEstablishedPeers(t *testing.T) {
	nodeA, err := NewNode("shutdown-a", "127.0.0.1:0", 0, "tip-a")
	if err != nil {
		t.Fatal(err)
	}
	nodeB, err := NewNode("shutdown-b", "127.0.0.1:0", 0, "tip-b")
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.Start(); err != nil {
		_ = nodeA.Stop()
		t.Fatal(err)
	}
	defer nodeB.Stop()

	if _, err := nodeA.Connect(nodeB.ListenAddress); err != nil {
		_ = nodeA.Stop()
		t.Fatal(err)
	}

	waitForPeerCount(t, nodeA, 1)
	waitForPeerCount(t, nodeB, 1)

	if err := nodeA.Stop(); err != nil {
		t.Fatal(err)
	}

	if nodeA.PeerCount() != 0 {
		t.Fatalf("stopped node retained %d peer(s)", nodeA.PeerCount())
	}

	// Closing node A's live connection must wake node B's read loop and remove
	// the remote peer instead of leaving a stale connection in its peer map.
	waitForPeerCount(t, nodeB, 0)
}

func TestStopIsIdempotent(t *testing.T) {
	node, err := NewNode("shutdown-idempotent", "127.0.0.1:0", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}

	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	if err := node.Stop(); err != nil {
		t.Fatalf("first stop failed: %v", err)
	}
	if err := node.Stop(); err != nil {
		t.Fatalf("second stop failed: %v", err)
	}
}
