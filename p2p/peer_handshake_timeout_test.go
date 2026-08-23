package p2p

import (
	"net"
	"testing"
	"time"
)

func TestStalledInboundHandshakeTimesOutAndReleasesSlot(t *testing.T) {
	node, err := NewNode("handshake-timeout", "127.0.0.1:0", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	conn, err := net.Dial("tcp", node.ListenAddress)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send nothing. The server must not let an unauthenticated client hold a
	// handshake goroutine and slot indefinitely.
	trackedDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(trackedDeadline) {
		node.mu.RLock()
		tracked := len(node.inboundHandshakeConns)
		node.mu.RUnlock()
		if tracked == 1 && len(node.inboundHandshakeSlots) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	node.mu.RLock()
	tracked := len(node.inboundHandshakeConns)
	node.mu.RUnlock()
	if tracked != 1 || len(node.inboundHandshakeSlots) != 1 {
		t.Fatalf("expected one active inbound handshake, tracked=%d slots=%d", tracked, len(node.inboundHandshakeSlots))
	}

	// Allow slightly more than the server's handshake deadline. A client-side
	// timeout means the server failed to close the stalled connection promptly.
	if err := conn.SetReadDeadline(time.Now().Add(DefaultDialTimeout + 2*time.Second)); err != nil {
		t.Fatal(err)
	}
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("stalled handshake connection remained readable past handshake deadline")
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		t.Fatal("stalled handshake was not closed by the server deadline")
	}

	releaseDeadline := time.Now().Add(time.Second)
	for time.Now().Before(releaseDeadline) {
		node.mu.RLock()
		tracked = len(node.inboundHandshakeConns)
		node.mu.RUnlock()
		if tracked == 0 && len(node.inboundHandshakeSlots) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	node.mu.RLock()
	tracked = len(node.inboundHandshakeConns)
	node.mu.RUnlock()
	t.Fatalf("stalled handshake resources were not released: tracked=%d slots=%d", tracked, len(node.inboundHandshakeSlots))
}
