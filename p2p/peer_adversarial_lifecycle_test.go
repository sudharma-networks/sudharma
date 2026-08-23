package p2p

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func waitForInboundHandshakes(t *testing.T, node *Node, expected int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		node.mu.RLock()
		got := len(node.inboundHandshakeConns)
		node.mu.RUnlock()
		if got == expected && len(node.inboundHandshakeSlots) == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	node.mu.RLock()
	got := len(node.inboundHandshakeConns)
	node.mu.RUnlock()
	t.Fatalf("inbound handshake state mismatch: conns=%d slots=%d expected=%d", got, len(node.inboundHandshakeSlots), expected)
}

func TestMalformedInboundHandshakeIsRejectedAndCleanedUp(t *testing.T) {
	node := startPeerSafetyTestNode(t, "malformed-handshake")
	conn, err := net.Dial("tcp", node.ListenAddress)
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	waitForInboundHandshakes(t, node, 1)
	if _, err := conn.Write([]byte("not-json\n")); err != nil { t.Fatal(err) }
	waitForInboundHandshakes(t, node, 0)
	if node.PeerCount() != 0 { t.Fatalf("malformed handshake admitted peer: %d", node.PeerCount()) }
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var b [1]byte
	if _, err := conn.Read(b[:]); err == nil { t.Fatal("malformed handshake connection remained open") }
}

func TestTruncatedInboundHandshakeIsRejectedAndCleanedUp(t *testing.T) {
	node := startPeerSafetyTestNode(t, "truncated-handshake")
	conn, err := net.Dial("tcp", node.ListenAddress)
	if err != nil { t.Fatal(err) }
	waitForInboundHandshakes(t, node, 1)
	if _, err := conn.Write([]byte(`{"type":"handshake"`)); err != nil { t.Fatal(err) }
	if tcp, ok := conn.(*net.TCPConn); ok { _ = tcp.CloseWrite() } else { _ = conn.Close() }
	waitForInboundHandshakes(t, node, 0)
	if node.PeerCount() != 0 { t.Fatalf("truncated handshake admitted peer: %d", node.PeerCount()) }
	_ = conn.Close()
}

func TestInboundHandshakeCapacityRecoversAfterMalformedFlood(t *testing.T) {
	node := startPeerSafetyTestNode(t, "handshake-recovery")
	for round := 0; round < 3; round++ {
		conns := make([]net.Conn, 0, MaxConcurrentInboundHandshakes)
		for i := 0; i < MaxConcurrentInboundHandshakes; i++ {
			conn, err := net.Dial("tcp", node.ListenAddress)
			if err != nil { t.Fatalf("round %d dial %d: %v", round, i, err) }
			conns = append(conns, conn)
		}
		waitForInboundHandshakes(t, node, MaxConcurrentInboundHandshakes)
		for i, conn := range conns {
			_, _ = fmt.Fprintf(conn, "bad-%d-%d\n", round, i)
		}
		for _, conn := range conns { _ = conn.Close() }
		waitForInboundHandshakes(t, node, 0)
	}
	if node.PeerCount() != 0 { t.Fatalf("malformed flood admitted peers: %d", node.PeerCount()) }
}

func TestRepeatedConnectDisconnectDoesNotLeaveStalePeers(t *testing.T) {
	nodeA := startPeerSafetyTestNode(t, "churn-a")
	nodeB := startPeerSafetyTestNode(t, "churn-b")
	for i := 0; i < 20; i++ {
		if _, err := nodeA.Connect(nodeB.ListenAddress); err != nil { t.Fatalf("connect iteration %d: %v", i, err) }
		waitForPeerCount(t, nodeA, 1)
		waitForPeerCount(t, nodeB, 1)
		nodeA.mu.RLock()
		peer := nodeA.peers["churn-b"]
		nodeA.mu.RUnlock()
		if peer == nil || peer.conn == nil { t.Fatalf("missing live peer at iteration %d", i) }
		_ = peer.conn.Close()
		waitForPeerCount(t, nodeA, 0)
		waitForPeerCount(t, nodeB, 0)
	}
}
