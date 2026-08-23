package p2p

import (
	"net"
	"testing"
	"time"
)

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

func TestStopClosesInFlightInboundHandshake(t *testing.T) {
	node, err := NewNode("shutdown-handshake", "127.0.0.1:0", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", node.ListenAddress)
	if err != nil {
		_ = node.Stop()
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		node.mu.RLock()
		tracked := len(node.inboundHandshakeConns)
		node.mu.RUnlock()
		if tracked == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	node.mu.RLock()
	tracked := len(node.inboundHandshakeConns)
	node.mu.RUnlock()
	if tracked != 1 {
		_ = node.Stop()
		t.Fatalf("expected one in-flight inbound handshake, got %d", tracked)
	}

	if err := node.Stop(); err != nil {
		t.Fatal(err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("in-flight handshake connection remained open after Stop")
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		t.Fatal("in-flight handshake connection was not closed promptly by Stop")
	}

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(node.inboundHandshakeSlots) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("inbound handshake slot remained occupied after Stop: %d", len(node.inboundHandshakeSlots))
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
