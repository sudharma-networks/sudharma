package p2p

import "testing"

func TestRecoveryCandidatesPreferValidatedHintsDeterministically(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	node.peers["peer-low"] = &PeerConnection{Info: PeerInfo{NodeID: "peer-low", Height: 10, TotalWork: "100"}}
	node.peers["peer-high-b"] = &PeerConnection{Info: PeerInfo{NodeID: "peer-high-b", Height: 20, TotalWork: "200"}}
	node.peers["peer-high-a"] = &PeerConnection{Info: PeerInfo{NodeID: "peer-high-a", Height: 20, TotalWork: "200"}}

	node.rewardPeer("peer-high-b", 5)

	candidates := node.recoveryCandidates()
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}
	if candidates[0].NodeID != "peer-high-b" {
		t.Fatalf("expected higher-score equal-work peer first, got %s", candidates[0].NodeID)
	}
	if candidates[1].NodeID != "peer-high-a" {
		t.Fatalf("expected second high-work peer next, got %s", candidates[1].NodeID)
	}
	if candidates[2].NodeID != "peer-low" {
		t.Fatalf("expected lower-work peer last, got %s", candidates[2].NodeID)
	}
}

func TestRecoveryCandidatesExcludeAvoidedAndInvalidPeers(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	node.peers["good"] = &PeerConnection{Info: PeerInfo{NodeID: "good", Height: 5, TotalWork: "50"}}
	node.peers["avoided"] = &PeerConnection{Info: PeerInfo{NodeID: "avoided", Height: 50, TotalWork: "500"}}
	node.peers["invalid"] = &PeerConnection{Info: PeerInfo{NodeID: "invalid", Height: 99, TotalWork: "not-a-number"}}

	node.penalizePeer("avoided", -PeerAvoidThreshold)

	candidates := node.recoveryCandidates()
	if len(candidates) != 1 || candidates[0].NodeID != "good" {
		t.Fatalf("expected only good peer, got %#v", candidates)
	}
	if got := node.PeerScore("invalid"); got != -PeerPenaltyInvalidData {
		t.Fatalf("expected invalid work advertisement penalty %d, got %d", -PeerPenaltyInvalidData, got)
	}
}

func TestRecoveryCandidatesAreBounded(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(node)

	for i := 0; i < MaxPartitionRecoveryPeers+5; i++ {
		node.peers[string(rune('a'+i))] = &PeerConnection{Info: PeerInfo{
			NodeID: string(rune('a' + i)), Height: uint64(i), TotalWork: "100",
		}}
	}

	if got := len(node.recoveryCandidates()); got != MaxPartitionRecoveryPeers {
		t.Fatalf("expected %d bounded candidates, got %d", MaxPartitionRecoveryPeers, got)
	}
}
