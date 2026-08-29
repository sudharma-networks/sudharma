package p2p

// removePeerConnection removes peer from the live peer map only when the
// currently stored connection is the same connection that is terminating.
// This prevents a stale read-loop cleanup from deleting a newer replacement
// connection that reused the same Node ID.
func (n *Node) removePeerConnection(peer *PeerConnection) {
	if n == nil || peer == nil || peer.Info.NodeID == "" {
		return
	}

	n.mu.Lock()
	current, ok := n.peers[peer.Info.NodeID]
	if !ok || current != peer {
		n.mu.Unlock()
		return
	}
	delete(n.peers, peer.Info.NodeID)
	n.mu.Unlock()

	// Closing after removal interrupts any read, write or keepalive goroutine
	// that already passed its map-membership check. Do not hold n.mu while the
	// transport is being closed.
	if peer.conn != nil {
		_ = peer.conn.Close()
	}
}
