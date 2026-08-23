package p2p

import (
	"net"
	"testing"
	"time"
)

func startPeerSafetyTestNode(
	t *testing.T,
	nodeID string,
) *Node {
	t.Helper()

	node, err := NewNode(
		nodeID,
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatalf(
			"failed creating node %s: %v",
			nodeID,
			err,
		)
	}

	if err := node.Start(); err != nil {
		t.Fatalf(
			"failed starting node %s: %v",
			nodeID,
			err,
		)
	}

	t.Cleanup(func() {
		_ = node.Stop()
	})

	return node
}

func waitForPeerCount(
	t *testing.T,
	node *Node,
	expected int,
) {
	t.Helper()

	deadline := time.Now().Add(
		2 * time.Second,
	)

	for time.Now().Before(deadline) {
		if node.PeerCount() == expected {
			return
		}

		time.Sleep(
			20 * time.Millisecond,
		)
	}

	t.Fatalf(
		"peer count mismatch: expected %d, got %d",
		expected,
		node.PeerCount(),
	)
}

func TestRejectDuplicatePeerConnection(
	t *testing.T,
) {
	nodeA := startPeerSafetyTestNode(
		t,
		"peer-safety-a",
	)

	nodeB := startPeerSafetyTestNode(
		t,
		"peer-safety-b",
	)

	_, err := nodeA.Connect(
		nodeB.ListenAddress,
	)
	if err != nil {
		t.Fatalf(
			"first connection failed: %v",
			err,
		)
	}

	waitForPeerCount(
		t,
		nodeA,
		1,
	)

	waitForPeerCount(
		t,
		nodeB,
		1,
	)

	_, err = nodeA.Connect(
		nodeB.ListenAddress,
	)

	if err == nil {
		t.Fatal(
			"expected duplicate peer connection to be rejected",
		)
	}

	if nodeA.PeerCount() != 1 {
		t.Fatalf(
			"duplicate connection changed node A peer count: %d",
			nodeA.PeerCount(),
		)
	}

	if nodeB.PeerCount() != 1 {
		t.Fatalf(
			"duplicate connection changed node B peer count: %d",
			nodeB.PeerCount(),
		)
	}
}

func TestRejectSelfConnection(
	t *testing.T,
) {
	node := startPeerSafetyTestNode(
		t,
		"peer-safety-self",
	)

	_, err := node.Connect(
		node.ListenAddress,
	)

	if err == nil {
		t.Fatal(
			"expected self connection to be rejected",
		)
	}

	time.Sleep(
		100 * time.Millisecond,
	)

	if node.PeerCount() != 0 {
		t.Fatalf(
			"self connection was stored; peer count: %d",
			node.PeerCount(),
		)
	}
}

func TestStorePeerRejectsEmptyNodeID(
	t *testing.T,
) {
	node, err := NewNode(
		"peer-safety-local",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatalf(
			"failed creating node: %v",
			err,
		)
	}

	localConn, remoteConn :=
		net.Pipe()

	defer localConn.Close()
	defer remoteConn.Close()

	peer := &PeerConnection{
		Info: PeerInfo{
			NodeID: "",
		},
		conn: localConn,
	}

	if node.storePeer(peer) {
		t.Fatal(
			"peer with empty node ID was accepted",
		)
	}

	if node.PeerCount() != 0 {
		t.Fatalf(
			"invalid peer changed peer count: %d",
			node.PeerCount(),
		)
	}
}

func TestStorePeerRejectsLocalNodeID(
	t *testing.T,
) {
	node, err := NewNode(
		"peer-safety-local",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatalf(
			"failed creating node: %v",
			err,
		)
	}

	localConn, remoteConn :=
		net.Pipe()

	defer localConn.Close()
	defer remoteConn.Close()

	peer := &PeerConnection{
		Info: PeerInfo{
			NodeID: node.NodeID,
		},
		conn: localConn,
	}

	if node.storePeer(peer) {
		t.Fatal(
			"peer using local node ID was accepted",
		)
	}

	if node.PeerCount() != 0 {
		t.Fatalf(
			"self peer changed peer count: %d",
			node.PeerCount(),
		)
	}
}
