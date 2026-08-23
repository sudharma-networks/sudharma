package p2p

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type recordingConn struct {
	mu     sync.Mutex
	writes [][]byte
}

func (c *recordingConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (c *recordingConn) Close() error                     { return nil }
func (c *recordingConn) LocalAddr() net.Addr              { return testAddr("local") }
func (c *recordingConn) RemoteAddr() net.Addr             { return testAddr("remote") }
func (c *recordingConn) SetDeadline(time.Time) error      { return nil }
func (c *recordingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *recordingConn) SetWriteDeadline(time.Time) error { return nil }
func (c *recordingConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, append([]byte(nil), p...))
	return len(p), nil
}
func (c *recordingConn) countWrites() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.writes)
}
func (c *recordingConn) lastWrite() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.writes) == 0 {
		return nil
	}
	return append([]byte(nil), c.writes[len(c.writes)-1]...)
}

type testAddr string

func (a testAddr) Network() string { return "test" }
func (a testAddr) String() string  { return string(a) }

func TestPartitionRecoveryRequestsIndependentNetworkGroups(t *testing.T) {
	n, err := NewNode("local", "127.0.0.1:9000", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(n)

	connA := &recordingConn{}
	connB := &recordingConn{}
	connC := &recordingConn{}

	n.peers["a"] = &PeerConnection{Info: PeerInfo{NodeID: "a"}, remoteAddress: "10.1.1.1:1000", conn: connA}
	n.peers["b"] = &PeerConnection{Info: PeerInfo{NodeID: "b"}, remoteAddress: "10.1.2.1:1000", conn: connB}
	n.peers["c"] = &PeerConnection{Info: PeerInfo{NodeID: "c"}, remoteAddress: "20.2.1.1:1000", conn: connC}

	requested, failed := n.requestPartitionRecoveryPeersOnce()
	if failed != 0 {
		t.Fatalf("expected no failed requests, got %d", failed)
	}
	if requested != 2 {
		t.Fatalf("expected two independent discovery requests, got %d", requested)
	}
	if connA.countWrites() != 1 || connC.countWrites() != 1 {
		t.Fatalf("expected one request to peers a and c, got a=%d c=%d", connA.countWrites(), connC.countWrites())
	}
	if connB.countWrites() != 0 {
		t.Fatalf("expected same-group peer b to be skipped, got %d writes", connB.countWrites())
	}

	expected, err := NewGetPeersMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(connA.lastWrite(), expected) || !bytes.Equal(connC.lastWrite(), expected) {
		t.Fatal("partition recovery did not send valid get-peers messages")
	}
}

func TestPartitionRecoveryCapsIndependentSources(t *testing.T) {
	n, err := NewNode("local", "127.0.0.1:9000", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(n)

	connections := make([]*recordingConn, 10)
	for i := 0; i < 10; i++ {
		conn := &recordingConn{}
		connections[i] = conn
		nodeID := string(rune('a' + i))
		address := net.JoinHostPort("host"+nodeID+".example", "1000")
		n.peers[nodeID] = &PeerConnection{Info: PeerInfo{NodeID: nodeID}, remoteAddress: address, conn: conn}
	}

	requested, failed := n.requestPartitionRecoveryPeersOnce()
	if failed != 0 {
		t.Fatalf("expected no failed requests, got %d", failed)
	}
	if requested != MaxPartitionRecoveryDiscoverySources {
		t.Fatalf("expected source cap %d, got %d", MaxPartitionRecoveryDiscoverySources, requested)
	}

	totalWrites := 0
	for _, conn := range connections {
		totalWrites += conn.countWrites()
	}
	if totalWrites != MaxPartitionRecoveryDiscoverySources {
		t.Fatalf("expected %d total requests, got %d", MaxPartitionRecoveryDiscoverySources, totalWrites)
	}
}

func TestRequestPartitionRecoveryPeersDoesNotStartLoopBeforeNodeStart(t *testing.T) {
	n, err := NewNode("local", "127.0.0.1:9000", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}

	n.RequestPartitionRecoveryPeers()
	if _, exists := nodePartitionRecoveryLoops.Load(n); exists {
		nodePartitionRecoveryLoops.Delete(n)
		t.Fatal("recovery loop should not start before the node listener is running")
	}
}
