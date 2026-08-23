package p2p

import "sync"

var nodePeerScorers sync.Map

func (n *Node) peerScorer() *PeerScorer {
	if n == nil {
		return NewPeerScorer()
	}
	if scorer, ok := nodePeerScorers.Load(n); ok {
		return scorer.(*PeerScorer)
	}
	scorer := NewPeerScorer()
	actual, _ := nodePeerScorers.LoadOrStore(n, scorer)
	return actual.(*PeerScorer)
}

// PeerScore returns this node's local reputation score for a remote peer.
func (n *Node) PeerScore(nodeID string) int {
	return n.peerScorer().Score(nodeID)
}

func (n *Node) rewardPeer(nodeID string, amount int) int {
	return n.peerScorer().Reward(nodeID, amount)
}

func (n *Node) penalizePeer(nodeID string, amount int) int {
	return n.peerScorer().Penalize(nodeID, amount)
}

func (n *Node) shouldAvoidPeer(nodeID string) bool {
	return n.peerScorer().ShouldAvoid(nodeID)
}

func clearNodePeerScorer(n *Node) {
	if n != nil {
		nodePeerScorers.Delete(n)
	}
}
