package p2p

// IsPeerConnected reports whether a peer with the given
// Node ID is currently active in the node's peer table.
func (n *Node) IsPeerConnected(
	nodeID string,
) bool {

	if nodeID == "" {
		return false
	}

	_, ok := n.Peer(
		nodeID,
	)

	return ok
}
