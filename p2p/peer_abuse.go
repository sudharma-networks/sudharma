package p2p

import "fmt"

// penalizePeerAndMaybeDisconnect applies a local reputation penalty and closes
// the peer connection once the disconnect threshold is reached.
//
// Peer reputation is a local networking policy only. It is never used as a
// consensus input.
func (n *Node) penalizePeerAndMaybeDisconnect(peer *PeerConnection, amount int, reason string) int {
	if n == nil || peer == nil || peer.Info.NodeID == "" || amount <= 0 {
		return PeerScoreInitial
	}

	score := n.penalizePeer(peer.Info.NodeID, amount)
	if score <= PeerDisconnectThreshold {
		fmt.Printf(
			"[PEERS] Disconnecting %s after reputation score reached %d (%s)\n",
			peer.Info.NodeID,
			score,
			reason,
		)
		if peer.conn != nil {
			_ = peer.conn.Close()
		}
	}

	return score
}

// rewardValidPeerMessage gives a small bounded reward for successfully
// validated protocol data. This keeps transient failures from permanently
// dominating a peer's reputation.
func (n *Node) rewardValidPeerMessage(peer *PeerConnection) int {
	if n == nil || peer == nil || peer.Info.NodeID == "" {
		return PeerScoreInitial
	}
	return n.rewardPeer(peer.Info.NodeID, PeerScoreGoodEvent)
}
