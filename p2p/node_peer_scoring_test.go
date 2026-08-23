package p2p

import "testing"

func TestNodePeerScoringIsLocalPerNode(t *testing.T) {
	n1, err := NewNode("node-a", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	n2, err := NewNode("node-b", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(n1)
	defer clearNodePeerScorer(n2)

	n1.penalizePeer("peer-x", 20)
	if got := n1.PeerScore("peer-x"); got != -20 {
		t.Fatalf("expected node-a score -20, got %d", got)
	}
	if got := n2.PeerScore("peer-x"); got != 0 {
		t.Fatalf("expected independent neutral score on node-b, got %d", got)
	}
}

func TestNodeShouldAvoidPeer(t *testing.T) {
	n, err := NewNode("node-a", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearNodePeerScorer(n)

	n.penalizePeer("peer-x", 30)
	if !n.shouldAvoidPeer("peer-x") {
		t.Fatal("expected peer-x to be avoided")
	}
}
