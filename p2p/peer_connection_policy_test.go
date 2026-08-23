package p2p

import (
	"fmt"
	"testing"
)

func TestStorePeerUsesActualRemoteAddressForDiversity(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	peers := []*PeerConnection{
		{Info: PeerInfo{NodeID: "peer-a", ListenAddress: "44.1.1.1:1000"}, remoteAddress: "10.20.1.1:2000"},
		{Info: PeerInfo{NodeID: "peer-b", ListenAddress: "55.1.1.1:1000"}, remoteAddress: "10.20.2.1:2000"},
		{Info: PeerInfo{NodeID: "peer-c", ListenAddress: "66.1.1.1:1000"}, remoteAddress: "10.20.3.1:2000"},
	}

	if !node.storePeer(peers[0]) || !node.storePeer(peers[1]) {
		t.Fatal("expected first two peers from the same /16 to be accepted")
	}
	if node.storePeer(peers[2]) {
		t.Fatal("expected third peer from the same actual /16 to be rejected despite different advertised address")
	}
}

func TestStorePeerEnforcesConnectionCapacity(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < MaxConnectedPeers; i++ {
		peer := &PeerConnection{
			Info:          PeerInfo{NodeID: fmt.Sprintf("peer-%d", i)},
			remoteAddress: fmt.Sprintf("10.%d.1.1:2000", i),
		}
		if !node.storePeer(peer) {
			t.Fatalf("expected peer %d to be accepted before capacity is reached", i)
		}
	}

	extra := &PeerConnection{
		Info:          PeerInfo{NodeID: "peer-extra"},
		remoteAddress: "172.31.1.1:2000",
	}
	if node.storePeer(extra) {
		t.Fatal("expected peer beyond connection capacity to be rejected")
	}
}
