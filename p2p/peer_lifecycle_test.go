package p2p

import (
	"net"
	"testing"
)

func TestRemovePeerConnectionDoesNotDeleteReplacement(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	oldPeer := &PeerConnection{Info: PeerInfo{NodeID: "peer-1"}}
	newPeer := &PeerConnection{Info: PeerInfo{NodeID: "peer-1"}}

	node.mu.Lock()
	node.peers["peer-1"] = newPeer
	node.mu.Unlock()

	// Simulate stale cleanup from an older read loop after a replacement
	// connection with the same Node ID has already been installed.
	node.removePeerConnection(oldPeer)

	node.mu.RLock()
	got := node.peers["peer-1"]
	node.mu.RUnlock()
	if got != newPeer {
		t.Fatal("stale peer cleanup removed or replaced the live connection")
	}
}

func TestRemovePeerConnectionRemovesExactConnection(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	peer := &PeerConnection{Info: PeerInfo{NodeID: "peer-1"}}
	node.mu.Lock()
	node.peers["peer-1"] = peer
	node.mu.Unlock()

	node.removePeerConnection(peer)
	if node.PeerCount() != 0 {
		t.Fatalf("expected peer map to be empty, got %d", node.PeerCount())
	}
}

func TestStopClosesAndClearsStoredPeers(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	local, remote := net.Pipe()
	defer remote.Close()
	peer := &PeerConnection{
		Info: PeerInfo{NodeID: "peer-1"},
		conn: local,
	}

	node.mu.Lock()
	node.peers["peer-1"] = peer
	node.mu.Unlock()

	if err := node.Stop(); err != nil {
		t.Fatal(err)
	}
	if node.PeerCount() != 0 {
		t.Fatalf("expected no stored peers after Stop, got %d", node.PeerCount())
	}

	buf := make([]byte, 1)
	if _, err := remote.Read(buf); err == nil {
		t.Fatal("expected remote side to observe closed peer connection")
	}
}
