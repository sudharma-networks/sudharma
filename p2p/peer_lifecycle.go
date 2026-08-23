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
	defer n.mu.Unlock()

	current, ok := n.peers[peer.Info.NodeID]
	if !ok || current != peer {
		return
	}
	delete(n.peers, peer.Info.NodeID)
}
