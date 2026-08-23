package p2p

const MaxConnectedPeers = 64

// peerNetworkAddress returns the address used for diversity decisions. The
// actual TCP remote address is preferred because handshake listen addresses
// are self-reported and therefore must not be trusted for eclipse protection.
func peerNetworkAddress(peer *PeerConnection) string {
	if peer == nil {
		return ""
	}
	if peer.remoteAddress != "" {
		return peer.remoteAddress
	}
	return peer.Info.ListenAddress
}

// canStorePeerLocked applies connection-capacity and network-diversity policy.
// The caller must hold n.mu.
func (n *Node) canStorePeerLocked(candidate *PeerConnection) bool {
	if n == nil || candidate == nil {
		return false
	}
	if len(n.peers) >= MaxConnectedPeers {
		return false
	}

	candidateAddress := peerNetworkAddress(candidate)
	if candidateAddress == "" {
		return false
	}

	addresses := make([]string, 0, len(n.peers))
	for _, peer := range n.peers {
		address := peerNetworkAddress(peer)
		if address != "" {
			addresses = append(addresses, address)
		}
	}

	return CanAddPeerFromNetworkGroup(addresses, candidateAddress)
}
