package p2p

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestLiveInboundHandshakeRejectsOversizedFrame(t *testing.T) {
	node, err := NewNode("oversized-handshake", "127.0.0.1:0", 0, "tip")
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
		t.Fatalf("expected one tracked inbound handshake, got %d", tracked)
	}

	payload := bytes.Repeat([]byte{'x'}, MaxPeerMessageBytes+1)
	_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	written, writeErr := io.Copy(conn, bytes.NewReader(payload))
	if writeErr != nil && written <= MaxPeerMessageBytes {
		t.Fatalf("connection failed before oversized frame crossed limit: wrote %d bytes: %v", written, writeErr)
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.CloseWrite()
	}

	// This window stays below DefaultDialTimeout so success demonstrates frame-
	// size rejection rather than the ordinary handshake timeout closing the peer.
	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		node.mu.RLock()
		tracked = len(node.inboundHandshakeConns)
		node.mu.RUnlock()
		if tracked == 0 && len(node.inboundHandshakeSlots) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	node.mu.RLock()
	tracked = len(node.inboundHandshakeConns)
	node.mu.RUnlock()
	if tracked != 0 {
		t.Fatalf("oversized handshake remained tracked after rejection: %d", tracked)
	}
	if got := len(node.inboundHandshakeSlots); got != 0 {
		t.Fatalf("oversized handshake retained concurrency slot: %d", got)
	}

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("oversized inbound handshake connection remained open")
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		t.Fatal("oversized inbound handshake was not closed promptly")
	}
}
