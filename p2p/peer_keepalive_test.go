package p2p

import (
	"bufio"
	"net"
	"testing"
	"time"
)

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
