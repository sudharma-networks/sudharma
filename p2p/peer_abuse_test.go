package p2p

import (
	"net"
	"testing"
)

func TestPenalizePeerAndMaybeDisconnect(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	local, remote := net.Pipe()
	defer remote.Close()

	peer := &PeerConnection{
		Info: PeerInfo{NodeID: "peer-a"},
		conn: local,
	}

	if score := node.penalizePeerAndMaybeDisconnect(peer, 30, "test"); score != -30 {
		t.Fatalf("expected -30 after first penalty, got %d", score)
	}

	if score := node.penalizePeerAndMaybeDisconnect(peer, 20, "test"); score != PeerDisconnectThreshold {
		t.Fatalf("expected disconnect threshold %d, got %d", PeerDisconnectThreshold, score)
	}

	buf := make([]byte, 1)
	if _, err := remote.Write([]byte{1}); err == nil {
		t.Fatal("expected remote write to fail after local connection was closed")
	}
	_ = buf
}

func TestRewardValidPeerMessage(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	peer := &PeerConnection{Info: PeerInfo{NodeID: "peer-a"}}
	if score := node.rewardValidPeerMessage(peer); score != PeerScoreGoodEvent {
		t.Fatalf("expected score %d, got %d", PeerScoreGoodEvent, score)
	}
}
