package p2p

import "testing"

func TestDiversifiedPeerOrderRoundRobinsNetworkGroups(t *testing.T) {
	peers := []KnownPeer{
		{NodeID: "a-1", Address: "10.1.1.1:1000"},
		{NodeID: "a-2", Address: "10.1.2.1:1000"},
		{NodeID: "b-1", Address: "20.2.1.1:1000"},
		{NodeID: "b-2", Address: "20.2.2.1:1000"},
		{NodeID: "c-1", Address: "30.3.1.1:1000"},
	}

	ordered := DiversifiedPeerOrder(peers)
	if len(ordered) != len(peers) {
		t.Fatalf("expected %d peers, got %d", len(peers), len(ordered))
	}

	firstGroups := make(map[string]struct{})
	for i := 0; i < 3; i++ {
		group, err := PeerNetworkGroup(ordered[i].Address)
		if err != nil {
			t.Fatal(err)
		}
		firstGroups[group] = struct{}{}
	}
	if len(firstGroups) != 3 {
		t.Fatalf("expected first reconnect round to cover 3 groups, got %d", len(firstGroups))
	}
}

func TestDiversifiedPeerOrderDropsDuplicatesAndInvalidEntries(t *testing.T) {
	peers := []KnownPeer{
		{NodeID: "a", Address: "10.1.1.1:1000"},
		{NodeID: "a", Address: "20.2.1.1:1000"},
		{NodeID: "b", Address: "10.1.1.1:1000"},
		{NodeID: "", Address: "30.3.1.1:1000"},
		{NodeID: "c", Address: "bad-address"},
	}

	ordered := DiversifiedPeerOrder(peers)
	if len(ordered) != 1 {
		t.Fatalf("expected one valid unique peer, got %d", len(ordered))
	}
	if ordered[0].NodeID != "a" {
		t.Fatalf("expected peer a, got %s", ordered[0].NodeID)
	}
}

func TestDiversifiedPeerOrderIsDeterministic(t *testing.T) {
	peers := []KnownPeer{
		{NodeID: "z", Address: "20.2.1.1:1000"},
		{NodeID: "b", Address: "10.1.2.1:1000"},
		{NodeID: "a", Address: "10.1.1.1:1000"},
	}

	first := DiversifiedPeerOrder(peers)
	second := DiversifiedPeerOrder(peers)
	if len(first) != len(second) {
		t.Fatal("deterministic order length mismatch")
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("order differs at index %d", i)
		}
	}
}
