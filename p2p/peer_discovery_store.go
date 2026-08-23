package p2p

import "sort"

// DiscoveredPeersSnapshot returns peers learned from peer discovery.
// These peers are not automatically connected here.
func (n *Node) DiscoveredPeersSnapshot() []KnownPeer {

	n.mu.RLock()
	defer n.mu.RUnlock()

	result := make(
		[]KnownPeer,
		0,
		len(n.discoveredPeers),
	)

	for _, peer := range n.discoveredPeers {
		result = append(
			result,
			peer,
		)
	}

	sort.Slice(
		result,
		func(i, j int) bool {
			return result[i].NodeID <
				result[j].NodeID
		},
	)

	return result
}

// mergeDiscoveredPeers safely adds newly learned peers.
func (n *Node) mergeDiscoveredPeers(
	peers []KnownPeer,
) int {

	filtered := n.filterDiscoveredPeers(
		peers,
	)

	if len(filtered) == 0 {
		return 0
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	added := 0

	for _, peer := range filtered {

		if peer.NodeID == n.NodeID ||
			peer.Address == n.ListenAddress {

			continue
		}

		if _, exists := n.discoveredPeers[peer.NodeID]; exists {
			continue
		}

		n.discoveredPeers[peer.NodeID] = peer
		added++
	}

	return added
}
