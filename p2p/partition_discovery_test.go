package p2p

import "testing"

func TestPartitionDiscoveryCandidatesUseDistinctNetworkGroups(t *testing.T) {
	n, err := NewNode("local", "127.0.0.1:9000", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}

	n.peers["a"] = &PeerConnection{Info: PeerInfo{NodeID: "a"}, remoteAddress: "10.1.1.1:1000"}
	n.peers["b"] = &PeerConnection{Info: PeerInfo{NodeID: "b"}, remoteAddress: "10.1.2.1:1000"}
	n.peers["c"] = &PeerConnection{Info: PeerInfo{NodeID: "c"}, remoteAddress: "20.2.1.1:1000"}

	selected := n.partitionDiscoveryCandidates()
	if len(selected) != 2 {
		t.Fatalf("expected two distinct network groups, got %d", len(selected))
	}

	groups := make(map[string]struct{})
	for _, peer := range selected {
		group, err := PeerNetworkGroup(peerNetworkAddress(peer))
		if err != nil {
			t.Fatal(err)
		}
		groups[group] = struct{}{}
	}
	if len(groups) != 2 {
		t.Fatalf("expected two unique groups, got %d", len(groups))
	}
}

func TestPartitionDiscoveryCandidatesSkipLowReputationPeers(t *testing.T) {
	n, err := NewNode("local", "127.0.0.1:9000", 0, "tip")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(n)

	n.peers["bad"] = &PeerConnection{Info: PeerInfo{NodeID: "bad"}, remoteAddress: "10.1.1.1:1000"}
	n.peers["good"] = &PeerConnection{Info: PeerInfo{NodeID: "good"}, remoteAddress: "20.2.1.1:1000"}
	n.penalizePeer("bad", 30)

	selected := n.partitionDiscoveryCandidates()
	if len(selected) != 1 {
		t.Fatalf("expected one eligible source, got %d", len(selected))
	}
	if selected[0].Info.NodeID != "good" {
		t.Fatalf("expected good peer, got %s", selected[0].Info.NodeID)
	}
}
