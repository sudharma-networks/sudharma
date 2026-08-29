package p2p

import (
	"bufio"
	"net"
	"sync"
	"testing"
	"time"
)

type blockingKeepaliveConn struct {
	mu            sync.Mutex
	writes        int
	secondStarted chan struct{}
	closed        chan struct{}
	secondOnce    sync.Once
	closeOnce     sync.Once
}

func newBlockingKeepaliveConn() *blockingKeepaliveConn {
	return &blockingKeepaliveConn{
		secondStarted: make(chan struct{}),
		closed:        make(chan struct{}),
	}
}

func (c *blockingKeepaliveConn) Read([]byte) (int, error) { return 0, net.ErrClosed }
func (c *blockingKeepaliveConn) Write(data []byte) (int, error) {
	c.mu.Lock()
	c.writes++
	writes := c.writes
	c.mu.Unlock()
	if writes == 1 {
		return len(data), nil
	}
	c.secondOnce.Do(func() { close(c.secondStarted) })
	<-c.closed
	return 0, net.ErrClosed
}
func (c *blockingKeepaliveConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}
func (c *blockingKeepaliveConn) LocalAddr() net.Addr              { return nil }
func (c *blockingKeepaliveConn) RemoteAddr() net.Addr             { return nil }
func (c *blockingKeepaliveConn) SetDeadline(time.Time) error      { return nil }
func (c *blockingKeepaliveConn) SetReadDeadline(time.Time) error  { return nil }
func (c *blockingKeepaliveConn) SetWriteDeadline(time.Time) error { return nil }

func TestPeerKeepaliveSendsPingAndStopsWhenPeerRemoved(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}

	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()

	peer := &PeerConnection{
		Info:   PeerInfo{NodeID: "remote", ListenAddress: "127.0.0.1:1", TotalWork: "1"},
		conn:   local,
		reader: bufio.NewReader(local),
	}
	node.mu.Lock()
	node.peers[peer.Info.NodeID] = peer
	node.mu.Unlock()

	done := make(chan struct{})
	go func() {
		node.keepPeerAliveWithInterval(peer, 5*time.Millisecond)
		close(done)
	}()

	if err := remote.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	data, err := readBoundedPeerMessage(bufio.NewReader(remote))
	if err != nil {
		t.Fatalf("read keepalive ping: %v", err)
	}
	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	ping, err := DecodePing(message)
	if err != nil {
		t.Fatalf("keepalive was not a valid ping: %v", err)
	}
	if ping.Nonce == 0 {
		t.Fatal("keepalive ping nonce must be non-zero")
	}

	node.removePeerConnection(peer)
	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("keepalive loop did not stop after peer removal")
	}
}

func TestPeerKeepaliveIntervalPrecedesReadIdleTimeout(t *testing.T) {
	if PeerKeepaliveInterval <= 0 || PeerKeepaliveInterval >= PeerReadIdleTimeout {
		t.Fatalf("invalid keepalive interval %s for idle timeout %s", PeerKeepaliveInterval, PeerReadIdleTimeout)
	}
}

func TestPeerRemovalInterruptsBlockedKeepaliveWrite(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}
	conn := newBlockingKeepaliveConn()
	defer conn.Close()
	peer := &PeerConnection{
		Info: PeerInfo{NodeID: "remote", ListenAddress: "127.0.0.1:1", TotalWork: "1"},
		conn: conn,
	}
	node.mu.Lock()
	node.peers[peer.Info.NodeID] = peer
	node.mu.Unlock()

	done := make(chan struct{})
	go func() {
		node.keepPeerAliveWithInterval(peer, time.Millisecond)
		close(done)
	}()

	select {
	case <-conn.secondStarted:
	case <-time.After(time.Second):
		t.Fatal("keepalive did not enter blocked second write")
	}
	node.removePeerConnection(peer)
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("peer removal did not interrupt blocked keepalive write")
	}
}
